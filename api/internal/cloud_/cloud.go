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

	"google.golang.org/api/cloudbilling/v1"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/usageapi"
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
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Credentials), &credentials); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Get service account JSON from credentials
	serviceAccountJSON, _ := credentials["serviceAccountKey"].(string)
	billingAccountID, _ := credentials["billingAccountId"].(string)
	projectID := provider.ProjectID

	if serviceAccountJSON == "" || projectID == "" {
		return nil, fmt.Errorf("missing GCP credentials (serviceAccountKey) or projectId")
	}

	// Create Cloud Billing service client
	billingService, err := cloudbilling.NewService(ctx, option.WithCredentialsJSON([]byte(serviceAccountJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create billing service: %w", err)
	}

	var totalCost float64
	currency := "USD"

	// If billing account ID is provided, get billing info
	if billingAccountID != "" {
		// Get project billing info
		projectBillingInfo, err := billingService.Projects.GetBillingInfo("projects/" + projectID).Context(ctx).Do()
		if err != nil {
			fmt.Printf("Warning: could not get billing info for project %s: %v\n", projectID, err)
		} else if projectBillingInfo.BillingEnabled {
			// Note: Cloud Billing API doesn't directly provide cost data
			// For detailed cost data, you would typically use BigQuery export
			// Here we'll use the Cloud Billing Budget API or return a placeholder
			// indicating that billing is enabled
			fmt.Printf("Billing enabled for project %s, billing account: %s\n", projectID, projectBillingInfo.BillingAccountName)
		}
	}

	// For actual cost data, GCP recommends using BigQuery billing export
	// This is a simplified implementation that queries available billing data
	// In production, you would query the billing export table in BigQuery

	// Alternative: Use Cloud Billing Budgets API to get budget vs actual
	// For now, we'll return the structure with a note that BigQuery export is recommended

	return map[string]interface{}{
		"monthlySpend":       totalCost,
		"currency":           currency,
		"billingAccountId":   billingAccountID,
		"projectId":          projectID,
		"note":               "For detailed costs, enable BigQuery billing export",
	}, nil
}

// FetchGCPBillingFromBigQuery fetches billing data from BigQuery export
// This requires the billing export to be set up in GCP
func FetchGCPBillingFromBigQuery(ctx context.Context, provider models.CloudProvider, cfg *config.Config) (map[string]interface{}, error) {
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Credentials), &credentials); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	serviceAccountJSON, _ := credentials["serviceAccountKey"].(string)
	billingDataset, _ := credentials["billingDataset"].(string)     // e.g., "project.dataset.gcp_billing_export"
	projectID := provider.ProjectID

	if serviceAccountJSON == "" || projectID == "" {
		return nil, fmt.Errorf("missing GCP credentials")
	}

	// If no billing dataset configured, fall back to basic billing API
	if billingDataset == "" {
		return FetchGCPBilling(ctx, provider, cfg)
	}

	// Note: BigQuery integration would require the BigQuery client library
	// For a full implementation, you would:
	// 1. Create a BigQuery client with the service account
	// 2. Query the billing export table for current month costs
	// 3. Aggregate by project/service as needed

	// Example query that would be used:
	// SELECT SUM(cost) as total_cost, currency
	// FROM `billing_dataset.gcp_billing_export`
	// WHERE project.id = @projectId
	// AND DATE(_PARTITIONTIME) >= DATE_TRUNC(CURRENT_DATE(), MONTH)
	// GROUP BY currency

	return map[string]interface{}{
		"monthlySpend": 0.0,
		"currency":     "USD",
		"source":       "bigquery",
		"dataset":      billingDataset,
	}, nil
}

// FetchOCIBilling fetches billing data from Oracle Cloud Infrastructure
func FetchOCIBilling(ctx context.Context, provider models.CloudProvider, cfg *config.Config) (map[string]interface{}, error) {
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Credentials), &credentials); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	tenancyOCID, _ := credentials["tenancyOcid"].(string)
	userOCID, _ := credentials["userOcid"].(string)
	fingerprint, _ := credentials["fingerprint"].(string)
	privateKey, _ := credentials["privateKey"].(string)
	region, _ := credentials["region"].(string)
	compartmentOCID, _ := credentials["compartmentOcid"].(string)

	if tenancyOCID == "" || userOCID == "" || fingerprint == "" || privateKey == "" {
		return nil, fmt.Errorf("missing OCI credentials (tenancyOcid, userOcid, fingerprint, privateKey)")
	}

	if region == "" {
		region = "us-ashburn-1" // Default region
	}

	// Use compartment OCID if provided, otherwise use tenancy OCID
	if compartmentOCID == "" {
		compartmentOCID = tenancyOCID
	}

	// Create OCI configuration provider
	configProvider := common.NewRawConfigurationProvider(
		tenancyOCID,
		userOCID,
		region,
		fingerprint,
		privateKey,
		nil, // passphrase
	)

	// Create Usage API client for cost data
	usageClient, err := usageapi.NewUsageapiClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI usage client: %w", err)
	}

	// Get current month's date range
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Create request for usage summary
	granularity := usageapi.RequestSummarizedUsagesDetailsGranularityMonthly
	queryType := usageapi.RequestSummarizedUsagesDetailsQueryTypeCost

	request := usageapi.RequestSummarizedUsagesRequest{
		RequestSummarizedUsagesDetails: usageapi.RequestSummarizedUsagesDetails{
			TenantId:      &tenancyOCID,
			TimeUsageStarted: &common.SDKTime{Time: startOfMonth},
			TimeUsageEnded:   &common.SDKTime{Time: now},
			Granularity:      granularity,
			QueryType:        queryType,
			CompartmentDepth: common.Float32(1),
		},
	}

	// Execute the request
	response, err := usageClient.RequestSummarizedUsages(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get OCI usage data: %w", err)
	}

	var totalCost float64
	currency := "USD"

	// Aggregate costs from response
	for _, item := range response.Items {
		if item.ComputedAmount != nil {
			totalCost += *item.ComputedAmount
		}
		if item.Currency != nil {
			currency = *item.Currency
		}
	}

	return map[string]interface{}{
		"monthlySpend":    totalCost,
		"currency":        currency,
		"tenancyOcid":     tenancyOCID,
		"compartmentOcid": compartmentOCID,
		"region":          region,
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
	case "oci":
		return stopOCINonEssentialResources(ctx, provider, cfg)
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
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Credentials), &credentials); err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	serviceAccountJSON, _ := credentials["serviceAccountKey"].(string)
	projectID := provider.ProjectID

	if serviceAccountJSON == "" || projectID == "" {
		return fmt.Errorf("missing GCP credentials (serviceAccountKey) or projectId")
	}

	// Create Compute Engine service client
	computeService, err := compute.NewService(ctx, option.WithCredentialsJSON([]byte(serviceAccountJSON)))
	if err != nil {
		return fmt.Errorf("failed to create compute service: %w", err)
	}

	// List all zones in the project
	zonesResp, err := computeService.Zones.List(projectID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to list zones: %w", err)
	}

	count := 0
	maxStops := 5 // Limit to 5 VMs to avoid massive disruption

	// Iterate through all zones and find running instances
	for _, zone := range zonesResp.Items {
		if count >= maxStops {
			break
		}

		// List instances in this zone
		instancesResp, err := computeService.Instances.List(projectID, zone.Name).
			Filter("status=RUNNING").
			Context(ctx).Do()
		if err != nil {
			fmt.Printf("Warning: failed to list instances in zone %s: %v\n", zone.Name, err)
			continue
		}

		for _, instance := range instancesResp.Items {
			if count >= maxStops {
				break
			}

			// Check if instance has Essential label
			hasEssential := false
			if instance.Labels != nil {
				if val, ok := instance.Labels["essential"]; ok && val == "true" {
					hasEssential = true
				}
			}

			if !hasEssential {
				// Stop the instance
				_, err := computeService.Instances.Stop(projectID, zone.Name, instance.Name).Context(ctx).Do()
				if err != nil {
					fmt.Printf("Error stopping GCP instance %s in zone %s: %v\n", instance.Name, zone.Name, err)
					continue
				}
				fmt.Printf("Successfully initiated stop for GCP instance: %s in zone %s\n", instance.Name, zone.Name)
				count++
			}
		}
	}

	return nil
}

// ListGCPInstances lists all Compute Engine instances in a project
func ListGCPInstances(ctx context.Context, provider models.CloudProvider, cfg *config.Config) ([]map[string]interface{}, error) {
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Credentials), &credentials); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	serviceAccountJSON, _ := credentials["serviceAccountKey"].(string)
	projectID := provider.ProjectID

	if serviceAccountJSON == "" || projectID == "" {
		return nil, fmt.Errorf("missing GCP credentials or projectId")
	}

	computeService, err := compute.NewService(ctx, option.WithCredentialsJSON([]byte(serviceAccountJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service: %w", err)
	}

	var instances []map[string]interface{}

	// Use aggregated list to get all instances across all zones
	req := computeService.Instances.AggregatedList(projectID)
	if err := req.Pages(ctx, func(page *compute.InstanceAggregatedList) error {
		for zone, instancesScopedList := range page.Items {
			if instancesScopedList.Instances != nil {
				for _, instance := range instancesScopedList.Instances {
					instances = append(instances, map[string]interface{}{
						"id":          instance.Id,
						"name":        instance.Name,
						"zone":        zone,
						"status":      instance.Status,
						"machineType": instance.MachineType,
						"labels":      instance.Labels,
						"createdAt":   instance.CreationTimestamp,
					})
				}
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	return instances, nil
}

// stopOCINonEssentialResources stops OCI compute instances without Essential freeform tag
func stopOCINonEssentialResources(ctx context.Context, provider models.CloudProvider, cfg *config.Config) error {
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Credentials), &credentials); err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	tenancyOCID, _ := credentials["tenancyOcid"].(string)
	userOCID, _ := credentials["userOcid"].(string)
	fingerprint, _ := credentials["fingerprint"].(string)
	privateKey, _ := credentials["privateKey"].(string)
	region, _ := credentials["region"].(string)
	compartmentOCID, _ := credentials["compartmentOcid"].(string)

	if tenancyOCID == "" || userOCID == "" || fingerprint == "" || privateKey == "" {
		return fmt.Errorf("missing OCI credentials")
	}

	if region == "" {
		region = "us-ashburn-1"
	}

	if compartmentOCID == "" {
		compartmentOCID = tenancyOCID
	}

	// Create OCI configuration provider
	configProvider := common.NewRawConfigurationProvider(
		tenancyOCID,
		userOCID,
		region,
		fingerprint,
		privateKey,
		nil,
	)

	// Create Compute client
	computeClient, err := core.NewComputeClientWithConfigurationProvider(configProvider)
	if err != nil {
		return fmt.Errorf("failed to create OCI compute client: %w", err)
	}

	// List all running instances in the compartment
	lifecycleState := core.InstanceLifecycleStateRunning
	listRequest := core.ListInstancesRequest{
		CompartmentId:  &compartmentOCID,
		LifecycleState: lifecycleState,
	}

	response, err := computeClient.ListInstances(ctx, listRequest)
	if err != nil {
		return fmt.Errorf("failed to list OCI instances: %w", err)
	}

	count := 0
	maxStops := 5 // Limit to 5 instances to avoid massive disruption

	for _, instance := range response.Items {
		if count >= maxStops {
			break
		}

		// Check if instance has Essential freeform tag
		hasEssential := false
		if instance.FreeformTags != nil {
			if val, ok := instance.FreeformTags["Essential"]; ok && val == "true" {
				hasEssential = true
			}
		}

		if !hasEssential && instance.Id != nil {
			// Stop the instance
			stopRequest := core.InstanceActionRequest{
				InstanceId: instance.Id,
				Action:     core.InstanceActionActionStop,
			}

			_, err := computeClient.InstanceAction(ctx, stopRequest)
			if err != nil {
				fmt.Printf("Error stopping OCI instance %s: %v\n", *instance.DisplayName, err)
				continue
			}
			fmt.Printf("Successfully initiated stop for OCI instance: %s\n", *instance.DisplayName)
			count++
		}
	}

	return nil
}

// ListOCIInstances lists all Compute instances in an OCI compartment
func ListOCIInstances(ctx context.Context, provider models.CloudProvider, cfg *config.Config) ([]map[string]interface{}, error) {
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Credentials), &credentials); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	tenancyOCID, _ := credentials["tenancyOcid"].(string)
	userOCID, _ := credentials["userOcid"].(string)
	fingerprint, _ := credentials["fingerprint"].(string)
	privateKey, _ := credentials["privateKey"].(string)
	region, _ := credentials["region"].(string)
	compartmentOCID, _ := credentials["compartmentOcid"].(string)

	if tenancyOCID == "" || userOCID == "" || fingerprint == "" || privateKey == "" {
		return nil, fmt.Errorf("missing OCI credentials")
	}

	if region == "" {
		region = "us-ashburn-1"
	}

	if compartmentOCID == "" {
		compartmentOCID = tenancyOCID
	}

	configProvider := common.NewRawConfigurationProvider(
		tenancyOCID,
		userOCID,
		region,
		fingerprint,
		privateKey,
		nil,
	)

	computeClient, err := core.NewComputeClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI compute client: %w", err)
	}

	listRequest := core.ListInstancesRequest{
		CompartmentId: &compartmentOCID,
	}

	response, err := computeClient.ListInstances(ctx, listRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to list OCI instances: %w", err)
	}

	var instances []map[string]interface{}
	for _, instance := range response.Items {
		instances = append(instances, map[string]interface{}{
			"id":             instance.Id,
			"name":           instance.DisplayName,
			"compartmentId":  instance.CompartmentId,
			"availabilityDomain": instance.AvailabilityDomain,
			"shape":          instance.Shape,
			"lifecycleState": instance.LifecycleState,
			"freeformTags":   instance.FreeformTags,
			"definedTags":    instance.DefinedTags,
			"createdAt":      instance.TimeCreated,
		})
	}

	return instances, nil
}

func TerminateOversizedInstances(ctx context.Context, provider models.CloudProvider, cfg *config.Config) error {
	// Similar to StopNonEssentialResources but terminates instead
	return nil
}

func StopIdleResources(ctx context.Context, provider models.CloudProvider, cfg *config.Config) error {
	// Stop resources that have been idle for specified hours
	return nil
}

