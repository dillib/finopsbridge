package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	config "finopsbridge/api/internal/config_"
	models "finopsbridge/api/internal/models_"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/consumption/armconsumption"
)

func FetchAWSBilling(ctx context.Context, provider models.CloudProvider, cfg *config.Config) (map[string]interface{}, error) {
	var credentials map[string]interface{}
	json.Unmarshal([]byte(provider.Credentials), &credentials)

	_, ok := credentials["roleArn"].(string)
	if !ok {
		return nil, fmt.Errorf("missing roleArn in credentials")
	}

	// Create AWS session with assumed role
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.AWSRegion),
	})
	if err != nil {
		return nil, err
	}

	// Use Cost Explorer to get billing data
	ce := costexplorer.New(sess)
	
	// Get current month's spend
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end := now

	result, err := ce.GetCostAndUsage(&costexplorer.GetCostAndUsageInput{
		TimePeriod: &costexplorer.DateInterval{
			Start: aws.String(start.Format("2006-01-02")),
			End:   aws.String(end.Format("2006-01-02")),
		},
		Granularity: aws.String("MONTHLY"),
		Metrics:     []*string{aws.String("BlendedCost")},
	})
	if err != nil {
		return nil, err
	}

	var monthlySpend float64
	if len(result.ResultsByTime) > 0 {
		if cost, exists := result.ResultsByTime[0].Total["BlendedCost"]; exists && cost.Amount != nil {
			fmt.Sscanf(*cost.Amount, "%f", &monthlySpend)
		}
	}

	return map[string]interface{}{
		"monthlySpend": monthlySpend,
		"currency":      "USD",
	}, nil
}

func FetchAzureBilling(ctx context.Context, provider models.CloudProvider, cfg *config.Config) (map[string]interface{}, error) {
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Credentials), &credentials); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	tenantID, _ := credentials["tenantId"].(string)
	clientID, _ := credentials["clientId"].(string)
	clientSecret, _ := credentials["clientSecret"].(string)
	subscriptionID := provider.SubscriptionID

	if tenantID == "" || clientID == "" || clientSecret == "" || subscriptionID == "" {
		return nil, fmt.Errorf("missing Azure credentials (tenantId, clientId, clientSecret) or subscriptionId")
	}

	// Create Azure credential using client secret
	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	// Create consumption client for cost data
	consumptionClient, err := armconsumption.NewUsageDetailsClient(cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumption client: %w", err)
	}

	// Get current month's date range
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Query scope for subscription-level costs
	scope := fmt.Sprintf("/subscriptions/%s", subscriptionID)

	// Build filter for current month
	filter := fmt.Sprintf("properties/usageStart ge '%s' and properties/usageEnd le '%s'",
		startOfMonth.Format("2006-01-02"),
		now.Format("2006-01-02"))

	var totalCost float64
	currency := "USD"

	// List usage details and aggregate costs
	pager := consumptionClient.NewListPager(scope, &armconsumption.UsageDetailsClientListOptions{
		Filter: &filter,
	})

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get usage details: %w", err)
		}

		for _, usage := range page.Value {
			// Handle legacy usage detail format
			if legacyUsage, ok := usage.(*armconsumption.LegacyUsageDetail); ok {
				if legacyUsage.Properties != nil && legacyUsage.Properties.CostInBillingCurrency != nil {
					totalCost += *legacyUsage.Properties.CostInBillingCurrency
				}
				if legacyUsage.Properties != nil && legacyUsage.Properties.BillingCurrency != nil {
					currency = *legacyUsage.Properties.BillingCurrency
				}
			}
			// Handle modern usage detail format
			if modernUsage, ok := usage.(*armconsumption.ModernUsageDetail); ok {
				if modernUsage.Properties != nil && modernUsage.Properties.CostInBillingCurrency != nil {
					totalCost += *modernUsage.Properties.CostInBillingCurrency
				}
				if modernUsage.Properties != nil && modernUsage.Properties.BillingCurrencyCode != nil {
					currency = *modernUsage.Properties.BillingCurrencyCode
				}
			}
		}
	}

	return map[string]interface{}{
		"monthlySpend": totalCost,
		"currency":     currency,
	}, nil
}

func FetchGCPBilling(ctx context.Context, provider models.CloudProvider, cfg *config.Config) (map[string]interface{}, error) {
	// GCP billing implementation
	// This would use GCP Billing API
	return map[string]interface{}{
		"monthlySpend": 0.0,
		"currency":     "USD",
	}, nil
}

func StopNonEssentialResources(ctx context.Context, provider models.CloudProvider, cfg *config.Config) error {
	switch provider.Type {
	case "aws":
		return stopAWSNonEssentialResources(ctx, provider, cfg)
	case "azure":
		return stopAzureNonEssentialResources(ctx, provider, cfg)
	case "gcp":
		return stopGCPNonEssentialResources(ctx, provider, cfg)
	}
	return nil
}

func stopAWSNonEssentialResources(ctx context.Context, provider models.CloudProvider, cfg *config.Config) error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.AWSRegion),
	})
	if err != nil {
		return err
	}

	ec2Svc := ec2.New(sess)
	
	// Find running instances without essential tags
	result, err := ec2Svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String("running")},
			},
		},
	})
	if err != nil {
		return err
	}

	// Stop instances (limit to 5 to avoid massive disruption)
	count := 0
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if count >= 5 {
				break
			}
			
			// Check if instance has essential tag
			hasEssential := false
			for _, tag := range instance.Tags {
				if *tag.Key == "Essential" && *tag.Value == "true" {
					hasEssential = true
					break
				}
			}

			if !hasEssential {
				_, err := ec2Svc.StopInstances(&ec2.StopInstancesInput{
					InstanceIds: []*string{instance.InstanceId},
				})
				if err != nil {
					fmt.Printf("Error stopping instance %s: %v\n", *instance.InstanceId, err)
				} else {
					count++
				}
			}
		}
	}

	return nil
}

func stopAzureNonEssentialResources(ctx context.Context, provider models.CloudProvider, cfg *config.Config) error {
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Credentials), &credentials); err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	tenantID, _ := credentials["tenantId"].(string)
	clientID, _ := credentials["clientId"].(string)
	clientSecret, _ := credentials["clientSecret"].(string)
	subscriptionID := provider.SubscriptionID

	if tenantID == "" || clientID == "" || clientSecret == "" || subscriptionID == "" {
		return fmt.Errorf("missing Azure credentials or subscriptionId")
	}

	// Create Azure credential
	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return fmt.Errorf("failed to create Azure credential: %w", err)
	}

	// Create VM client
	vmClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create VM client: %w", err)
	}

	// List all VMs in the subscription
	pager := vmClient.NewListAllPager(nil)

	count := 0
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list VMs: %w", err)
		}

		for _, vm := range page.Value {
			if count >= 5 {
				// Limit to 5 VMs to avoid massive disruption
				break
			}

			// Check if VM has Essential tag
			hasEssential := false
			if vm.Tags != nil {
				if val, ok := vm.Tags["Essential"]; ok && val != nil && *val == "true" {
					hasEssential = true
				}
			}

			if !hasEssential && vm.Name != nil && vm.ID != nil {
				// Extract resource group from VM ID
				// VM ID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Compute/virtualMachines/{name}
				resourceGroup := extractResourceGroupFromID(*vm.ID)
				if resourceGroup == "" {
					fmt.Printf("Could not extract resource group from VM ID: %s\n", *vm.ID)
					continue
				}

				// Deallocate (stop) the VM
				poller, err := vmClient.BeginDeallocate(ctx, resourceGroup, *vm.Name, nil)
				if err != nil {
					fmt.Printf("Error stopping Azure VM %s: %v\n", *vm.Name, err)
					continue
				}

				// Wait for the operation to complete (with timeout)
				_, err = poller.PollUntilDone(ctx, nil)
				if err != nil {
					fmt.Printf("Error waiting for VM %s to stop: %v\n", *vm.Name, err)
				} else {
					fmt.Printf("Successfully stopped Azure VM: %s\n", *vm.Name)
					count++
				}
			}
		}
	}

	return nil
}

// extractResourceGroupFromID extracts the resource group name from an Azure resource ID
func extractResourceGroupFromID(resourceID string) string {
	// ID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/...
	parts := splitAzureResourceID(resourceID)
	for i, part := range parts {
		if part == "resourceGroups" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func splitAzureResourceID(id string) []string {
	var parts []string
	current := ""
	for _, char := range id {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func stopGCPNonEssentialResources(ctx context.Context, provider models.CloudProvider, cfg *config.Config) error {
	// GCP implementation
	return nil
}

func TerminateOversizedInstances(ctx context.Context, provider models.CloudProvider, cfg *config.Config) error {
	// Similar to StopNonEssentialResources but terminates instead
	return nil
}

func StopIdleResources(ctx context.Context, provider models.CloudProvider, cfg *config.Config) error {
	// Stop resources that have been idle for specified hours
	return nil
}

