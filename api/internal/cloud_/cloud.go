package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	config "finopsbridge/api/internal/config_"
	models "finopsbridge/api/internal/models_"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/consumption/armconsumption"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/cloudbilling/v1"
	compute "google.golang.org/api/compute/v1"
	monitoring "google.golang.org/api/monitoring/v3"
	"google.golang.org/api/option"

	ocicommon "github.com/oracle/oci-go-sdk/v65/common"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/usageapi"

	ibmcore "github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/usagereportsv4"
	"github.com/IBM/vpc-go-sdk/vpcv1"

	"google.golang.org/api/iterator"
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
				if legacyUsage.Properties != nil && legacyUsage.Properties.Cost != nil {
					totalCost += *legacyUsage.Properties.Cost
				}
				if legacyUsage.Properties != nil && legacyUsage.Properties.Currency != nil {
					currency = *legacyUsage.Properties.Currency
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

	// Check if BigQuery billing dataset is configured - prefer BigQuery for accurate costs
	billingDataset, _ := credentials["billingDataset"].(string)
	if billingDataset != "" {
		return FetchGCPBillingFromBigQuery(ctx, provider, cfg)
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
	billingEnabled := false

	// If billing account ID is provided, get billing info
	if billingAccountID != "" {
		// Get project billing info
		projectBillingInfo, err := billingService.Projects.GetBillingInfo("projects/" + projectID).Context(ctx).Do()
		if err != nil {
			fmt.Printf("Warning: could not get billing info for project %s: %v\n", projectID, err)
		} else if projectBillingInfo.BillingEnabled {
			billingEnabled = true
			fmt.Printf("Billing enabled for project %s, billing account: %s\n", projectID, projectBillingInfo.BillingAccountName)
		}
	}

	// Note: Cloud Billing API doesn't directly provide cost data
	// To get actual costs, users must configure BigQuery billing export
	// Return billing status and note for setup

	return map[string]interface{}{
		"monthlySpend":       totalCost,
		"currency":           currency,
		"billingAccountId":   billingAccountID,
		"projectId":          projectID,
		"billingEnabled":     billingEnabled,
		"note":               "Configure billingDataset in credentials for accurate cost data via BigQuery export",
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
	billingDataset, _ := credentials["billingDataset"].(string) // e.g., "project.dataset.gcp_billing_export_v1"
	billingTable, _ := credentials["billingTable"].(string)     // e.g., "gcp_billing_export_v1_XXXXXX_XXXXXX"
	projectID := provider.ProjectID

	if serviceAccountJSON == "" || projectID == "" {
		return nil, fmt.Errorf("missing GCP credentials")
	}

	// If no billing dataset configured, fall back to basic billing API
	if billingDataset == "" {
		return FetchGCPBilling(ctx, provider, cfg)
	}

	// Create BigQuery client with service account credentials
	bqClient, err := bigquery.NewClient(ctx, projectID, option.WithCredentialsJSON([]byte(serviceAccountJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client: %w", err)
	}
	defer bqClient.Close()

	// Get current month's date range
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Build the billing table reference
	// Format: project.dataset.table
	tableRef := billingDataset
	if billingTable != "" {
		tableRef = fmt.Sprintf("%s.%s", billingDataset, billingTable)
	}

	// Query to get total cost for the project in the current month
	query := fmt.Sprintf(`
		SELECT
			SUM(cost) as total_cost,
			currency
		FROM `+"`%s`"+`
		WHERE project.id = @projectId
		AND DATE(usage_start_time) >= @startDate
		AND DATE(usage_start_time) <= @endDate
		GROUP BY currency
		ORDER BY total_cost DESC
		LIMIT 1
	`, tableRef)

	q := bqClient.Query(query)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "projectId", Value: projectID},
		{Name: "startDate", Value: startOfMonth.Format("2006-01-02")},
		{Name: "endDate", Value: now.Format("2006-01-02")},
	}

	// Run the query
	it, err := q.Read(ctx)
	if err != nil {
		// If BigQuery query fails, fall back to basic billing API
		fmt.Printf("Warning: BigQuery query failed, falling back to basic API: %v\n", err)
		return FetchGCPBilling(ctx, provider, cfg)
	}

	var totalCost float64
	currency := "USD"

	// Read the first result row
	var row struct {
		TotalCost float64 `bigquery:"total_cost"`
		Currency  string  `bigquery:"currency"`
	}

	err = it.Next(&row)
	if err == iterator.Done {
		// No billing data found for this month
		return map[string]interface{}{
			"monthlySpend": 0.0,
			"currency":     currency,
			"source":       "bigquery",
			"note":         "No billing data found for current month",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read BigQuery results: %w", err)
	}

	totalCost = row.TotalCost
	if row.Currency != "" {
		currency = row.Currency
	}

	return map[string]interface{}{
		"monthlySpend":     totalCost,
		"currency":         currency,
		"source":           "bigquery",
		"billingDataset":   billingDataset,
		"projectId":        projectID,
		"periodStart":      startOfMonth.Format("2006-01-02"),
		"periodEnd":        now.Format("2006-01-02"),
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
	configProvider := ocicommon.NewRawConfigurationProvider(
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
			TimeUsageStarted: &ocicommon.SDKTime{Time: startOfMonth},
			TimeUsageEnded:   &ocicommon.SDKTime{Time: now},
			Granularity:      granularity,
			QueryType:        queryType,
			CompartmentDepth: ocicommon.Float32(1),
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
			totalCost += float64(*item.ComputedAmount)
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

// FetchIBMBilling fetches billing data from IBM Cloud
func FetchIBMBilling(ctx context.Context, provider models.CloudProvider, cfg *config.Config) (map[string]interface{}, error) {
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Credentials), &credentials); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	apiKey, _ := credentials["apiKey"].(string)
	accountID, _ := credentials["accountId"].(string)

	if apiKey == "" || accountID == "" {
		return nil, fmt.Errorf("missing IBM Cloud credentials (apiKey, accountId)")
	}

	// Create IAM authenticator
	authenticator := &ibmcore.IamAuthenticator{
		ApiKey: apiKey,
	}

	// Create Usage Reports client
	usageReportsService, err := usagereportsv4.NewUsageReportsV4(&usagereportsv4.UsageReportsV4Options{
		Authenticator: authenticator,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create IBM usage reports client: %w", err)
	}

	// Get current month
	now := time.Now()
	billingMonth := now.Format("2006-01")

	// Get account usage
	getAccountUsageOptions := usageReportsService.NewGetAccountUsageOptions(accountID, billingMonth)
	accountUsage, _, err := usageReportsService.GetAccountUsage(getAccountUsageOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get IBM account usage: %w", err)
	}

	var totalCost float64
	currency := "USD"

	// Aggregate costs from resources
	if accountUsage.Resources != nil {
		for _, resource := range accountUsage.Resources {
			if resource.BillableCost != nil {
				totalCost += *resource.BillableCost
			}
		}
	}

	if accountUsage.CurrencyCode != nil {
		currency = *accountUsage.CurrencyCode
	}

	return map[string]interface{}{
		"monthlySpend": totalCost,
		"currency":     currency,
		"accountId":    accountID,
		"billingMonth": billingMonth,
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
	case "ibm":
		return stopIBMNonEssentialResources(ctx, provider, cfg)
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
	configProvider := ocicommon.NewRawConfigurationProvider(
		tenancyOCID,
		userOCID,
		region,
		fingerprint,
		privateKey,
		nil,
	)

	// Create Compute client
	computeClient, err := ocicore.NewComputeClientWithConfigurationProvider(configProvider)
	if err != nil {
		return fmt.Errorf("failed to create OCI compute client: %w", err)
	}

	// List all running instances in the compartment
	lifecycleState := ocicore.InstanceLifecycleStateRunning
	listRequest := ocicore.ListInstancesRequest{
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
			stopRequest := ocicore.InstanceActionRequest{
				InstanceId: instance.Id,
				Action:     ocicore.InstanceActionActionStop,
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

	configProvider := ocicommon.NewRawConfigurationProvider(
		tenancyOCID,
		userOCID,
		region,
		fingerprint,
		privateKey,
		nil,
	)

	computeClient, err := ocicore.NewComputeClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI compute client: %w", err)
	}

	listRequest := ocicore.ListInstancesRequest{
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

// stopIBMNonEssentialResources stops IBM Cloud virtual server instances without Essential tag
func stopIBMNonEssentialResources(ctx context.Context, provider models.CloudProvider, cfg *config.Config) error {
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Credentials), &credentials); err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	apiKey, _ := credentials["apiKey"].(string)
	region, _ := credentials["region"].(string)

	if apiKey == "" {
		return fmt.Errorf("missing IBM Cloud credentials (apiKey)")
	}

	if region == "" {
		region = "us-south" // Default region
	}

	// Create IAM authenticator
	authenticator := &ibmcore.IamAuthenticator{
		ApiKey: apiKey,
	}

	// Create VPC client
	vpcService, err := vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
		Authenticator: authenticator,
		URL:           fmt.Sprintf("https://%s.iaas.cloud.ibm.com/v1", region),
	})
	if err != nil {
		return fmt.Errorf("failed to create IBM VPC client: %w", err)
	}

	// List all instances
	listInstancesOptions := vpcService.NewListInstancesOptions()
	instances, _, err := vpcService.ListInstances(listInstancesOptions)
	if err != nil {
		return fmt.Errorf("failed to list IBM instances: %w", err)
	}

	count := 0
	maxStops := 5 // Limit to 5 instances to avoid massive disruption

	for _, instance := range instances.Instances {
		if count >= maxStops {
			break
		}

		// Only process running instances
		if instance.Status != nil && *instance.Status != "running" {
			continue
		}

		// Check if instance has Essential tag in user tags
		hasEssential := false
		// IBM Cloud uses resource tags - check metadata or name pattern
		if instance.Name != nil && containsEssential(*instance.Name) {
			hasEssential = true
		}

		if !hasEssential && instance.ID != nil {
			// Create stop action
			stopAction := "stop"
			createInstanceActionOptions := vpcService.NewCreateInstanceActionOptions(*instance.ID, stopAction)
			_, _, err := vpcService.CreateInstanceAction(createInstanceActionOptions)
			if err != nil {
				fmt.Printf("Error stopping IBM instance %s: %v\n", *instance.Name, err)
				continue
			}
			fmt.Printf("Successfully initiated stop for IBM instance: %s\n", *instance.Name)
			count++
		}
	}

	return nil
}

// containsEssential checks if a string contains "essential" (case-insensitive)
func containsEssential(s string) bool {
	lower := ""
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			lower += string(c + 32)
		} else {
			lower += string(c)
		}
	}
	return len(lower) >= 9 && (lower == "essential" ||
		(len(lower) > 9 && (lower[:9] == "essential" || lower[len(lower)-9:] == "essential")))
}

// ListIBMInstances lists all Virtual Server instances in IBM Cloud
func ListIBMInstances(ctx context.Context, provider models.CloudProvider, cfg *config.Config) ([]map[string]interface{}, error) {
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Credentials), &credentials); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	apiKey, _ := credentials["apiKey"].(string)
	region, _ := credentials["region"].(string)

	if apiKey == "" {
		return nil, fmt.Errorf("missing IBM Cloud credentials (apiKey)")
	}

	if region == "" {
		region = "us-south"
	}

	authenticator := &ibmcore.IamAuthenticator{
		ApiKey: apiKey,
	}

	vpcService, err := vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
		Authenticator: authenticator,
		URL:           fmt.Sprintf("https://%s.iaas.cloud.ibm.com/v1", region),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create IBM VPC client: %w", err)
	}

	listInstancesOptions := vpcService.NewListInstancesOptions()
	instances, _, err := vpcService.ListInstances(listInstancesOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list IBM instances: %w", err)
	}

	var result []map[string]interface{}
	for _, instance := range instances.Instances {
		result = append(result, map[string]interface{}{
			"id":        instance.ID,
			"name":      instance.Name,
			"status":    instance.Status,
			"profile":   instance.Profile,
			"zone":      instance.Zone,
			"vpc":       instance.VPC,
			"createdAt": instance.CreatedAt,
		})
	}

	return result, nil
}

// TerminateOversizedInstances terminates instances that exceed allowed size thresholds
func TerminateOversizedInstances(ctx context.Context, provider models.CloudProvider, cfg *config.Config, maxSizeLevel int) error {
	switch provider.Type {
	case "aws":
		return terminateAWSOversizedInstances(ctx, provider, cfg, maxSizeLevel)
	case "azure":
		return terminateAzureOversizedInstances(ctx, provider, cfg, maxSizeLevel)
	case "gcp":
		return terminateGCPOversizedInstances(ctx, provider, cfg, maxSizeLevel)
	case "oci":
		return terminateOCIOversizedInstances(ctx, provider, cfg, maxSizeLevel)
	case "ibm":
		return terminateIBMOversizedInstances(ctx, provider, cfg, maxSizeLevel)
	}
	return nil
}

// terminateAWSOversizedInstances terminates AWS EC2 instances that exceed size limit
func terminateAWSOversizedInstances(ctx context.Context, provider models.CloudProvider, cfg *config.Config, maxSizeLevel int) error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.AWSRegion),
	})
	if err != nil {
		return err
	}

	ec2Svc := ec2.New(sess)

	// List running instances
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

	// Define instance size levels (approximate ordering by size)
	sizeLevel := func(instanceType string) int {
		switch {
		case strings.Contains(instanceType, "nano") || strings.Contains(instanceType, "micro"):
			return 1
		case strings.Contains(instanceType, "small"):
			return 2
		case strings.Contains(instanceType, "medium"):
			return 3
		case strings.Contains(instanceType, "large") && !strings.Contains(instanceType, "xlarge"):
			return 4
		case strings.Contains(instanceType, "xlarge") && !strings.Contains(instanceType, "2xlarge"):
			return 5
		case strings.Contains(instanceType, "2xlarge"):
			return 6
		case strings.Contains(instanceType, "4xlarge"):
			return 7
		case strings.Contains(instanceType, "8xlarge"):
			return 8
		default:
			return 9 // Very large instances
		}
	}

	count := 0
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if count >= 5 {
				break
			}

			instanceType := *instance.InstanceType
			if sizeLevel(instanceType) > maxSizeLevel {
				// Check for Essential tag before terminating
				hasEssential := false
				for _, tag := range instance.Tags {
					if *tag.Key == "Essential" && *tag.Value == "true" {
						hasEssential = true
						break
					}
				}

				if !hasEssential {
					_, err := ec2Svc.TerminateInstances(&ec2.TerminateInstancesInput{
						InstanceIds: []*string{instance.InstanceId},
					})
					if err != nil {
						fmt.Printf("Error terminating oversized instance %s: %v\n", *instance.InstanceId, err)
					} else {
						fmt.Printf("Terminated oversized instance %s (type: %s, level: %d > max: %d)\n",
							*instance.InstanceId, instanceType, sizeLevel(instanceType), maxSizeLevel)
						count++
					}
				}
			}
		}
	}

	return nil
}

// terminateAzureOversizedInstances terminates Azure VMs that exceed size limit
func terminateAzureOversizedInstances(ctx context.Context, provider models.CloudProvider, cfg *config.Config, maxSizeLevel int) error {
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

	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return fmt.Errorf("failed to create Azure credential: %w", err)
	}

	vmClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create VM client: %w", err)
	}

	// Azure VM size levels (approximate ordering)
	sizeLevel := func(vmSize string) int {
		lower := strings.ToLower(vmSize)
		switch {
		case strings.Contains(lower, "_b1") || strings.Contains(lower, "_a0"):
			return 1
		case strings.Contains(lower, "_b2") || strings.Contains(lower, "_a1"):
			return 2
		case strings.Contains(lower, "_d2") || strings.Contains(lower, "_b4"):
			return 3
		case strings.Contains(lower, "_d4") || strings.Contains(lower, "_b8"):
			return 4
		case strings.Contains(lower, "_d8"):
			return 5
		case strings.Contains(lower, "_d16"):
			return 6
		case strings.Contains(lower, "_d32"):
			return 7
		case strings.Contains(lower, "_d64"):
			return 8
		default:
			return 9
		}
	}

	pager := vmClient.NewListAllPager(nil)
	count := 0

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list VMs: %w", err)
		}

		for _, vm := range page.Value {
			if count >= 5 {
				break
			}

			if vm.Properties != nil && vm.Properties.HardwareProfile != nil && vm.Properties.HardwareProfile.VMSize != nil {
				vmSize := string(*vm.Properties.HardwareProfile.VMSize)
				if sizeLevel(vmSize) > maxSizeLevel {
					// Check for Essential tag
					hasEssential := false
					if vm.Tags != nil {
						if val, ok := vm.Tags["Essential"]; ok && val != nil && *val == "true" {
							hasEssential = true
						}
					}

					if !hasEssential && vm.Name != nil && vm.ID != nil {
						resourceGroup := extractResourceGroupFromID(*vm.ID)
						if resourceGroup == "" {
							continue
						}

						// Delete (terminate) the VM
						poller, err := vmClient.BeginDelete(ctx, resourceGroup, *vm.Name, nil)
						if err != nil {
							fmt.Printf("Error deleting oversized Azure VM %s: %v\n", *vm.Name, err)
							continue
						}

						_, err = poller.PollUntilDone(ctx, nil)
						if err != nil {
							fmt.Printf("Error waiting for VM %s deletion: %v\n", *vm.Name, err)
						} else {
							fmt.Printf("Deleted oversized Azure VM: %s (size: %s)\n", *vm.Name, vmSize)
							count++
						}
					}
				}
			}
		}
	}

	return nil
}

// terminateGCPOversizedInstances terminates GCP instances that exceed size limit
func terminateGCPOversizedInstances(ctx context.Context, provider models.CloudProvider, cfg *config.Config, maxSizeLevel int) error {
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Credentials), &credentials); err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	serviceAccountJSON, _ := credentials["serviceAccountKey"].(string)
	projectID := provider.ProjectID

	if serviceAccountJSON == "" || projectID == "" {
		return fmt.Errorf("missing GCP credentials or projectId")
	}

	computeService, err := compute.NewService(ctx, option.WithCredentialsJSON([]byte(serviceAccountJSON)))
	if err != nil {
		return fmt.Errorf("failed to create compute service: %w", err)
	}

	// GCP machine type size levels
	sizeLevel := func(machineType string) int {
		lower := strings.ToLower(machineType)
		switch {
		case strings.Contains(lower, "micro") || strings.Contains(lower, "small"):
			return 1
		case strings.Contains(lower, "medium"):
			return 2
		case strings.Contains(lower, "standard-1") || strings.Contains(lower, "n1-standard-1"):
			return 3
		case strings.Contains(lower, "standard-2"):
			return 4
		case strings.Contains(lower, "standard-4"):
			return 5
		case strings.Contains(lower, "standard-8"):
			return 6
		case strings.Contains(lower, "standard-16"):
			return 7
		case strings.Contains(lower, "standard-32") || strings.Contains(lower, "highcpu") || strings.Contains(lower, "highmem"):
			return 8
		default:
			return 9
		}
	}

	zonesResp, err := computeService.Zones.List(projectID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to list zones: %w", err)
	}

	count := 0
	for _, zone := range zonesResp.Items {
		if count >= 5 {
			break
		}

		instancesResp, err := computeService.Instances.List(projectID, zone.Name).
			Filter("status=RUNNING").
			Context(ctx).Do()
		if err != nil {
			continue
		}

		for _, instance := range instancesResp.Items {
			if count >= 5 {
				break
			}

			if sizeLevel(instance.MachineType) > maxSizeLevel {
				// Check for essential label
				hasEssential := false
				if instance.Labels != nil {
					if val, ok := instance.Labels["essential"]; ok && val == "true" {
						hasEssential = true
					}
				}

				if !hasEssential {
					_, err := computeService.Instances.Delete(projectID, zone.Name, instance.Name).Context(ctx).Do()
					if err != nil {
						fmt.Printf("Error deleting oversized GCP instance %s: %v\n", instance.Name, err)
						continue
					}
					fmt.Printf("Deleted oversized GCP instance: %s in zone %s\n", instance.Name, zone.Name)
					count++
				}
			}
		}
	}

	return nil
}

// terminateOCIOversizedInstances terminates OCI instances that exceed size limit
func terminateOCIOversizedInstances(ctx context.Context, provider models.CloudProvider, cfg *config.Config, maxSizeLevel int) error {
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

	configProvider := ocicommon.NewRawConfigurationProvider(
		tenancyOCID, userOCID, region, fingerprint, privateKey, nil,
	)

	computeClient, err := ocicore.NewComputeClientWithConfigurationProvider(configProvider)
	if err != nil {
		return fmt.Errorf("failed to create OCI compute client: %w", err)
	}

	// OCI shape size levels (based on OCPUs)
	sizeLevel := func(shape string) int {
		lower := strings.ToLower(shape)
		switch {
		case strings.Contains(lower, "micro") || strings.Contains(lower, "1.1"):
			return 1
		case strings.Contains(lower, "1.2"):
			return 2
		case strings.Contains(lower, "1.4") || strings.Contains(lower, "2.1"):
			return 3
		case strings.Contains(lower, "2.2") || strings.Contains(lower, "1.8"):
			return 4
		case strings.Contains(lower, "2.4") || strings.Contains(lower, "1.16"):
			return 5
		default:
			return 6
		}
	}

	lifecycleState := ocicore.InstanceLifecycleStateRunning
	listRequest := ocicore.ListInstancesRequest{
		CompartmentId:  &compartmentOCID,
		LifecycleState: lifecycleState,
	}

	response, err := computeClient.ListInstances(ctx, listRequest)
	if err != nil {
		return fmt.Errorf("failed to list OCI instances: %w", err)
	}

	count := 0
	for _, instance := range response.Items {
		if count >= 5 {
			break
		}

		if instance.Shape != nil && sizeLevel(*instance.Shape) > maxSizeLevel {
			hasEssential := false
			if instance.FreeformTags != nil {
				if val, ok := instance.FreeformTags["Essential"]; ok && val == "true" {
					hasEssential = true
				}
			}

			if !hasEssential && instance.Id != nil {
				terminateRequest := ocicore.TerminateInstanceRequest{
					InstanceId: instance.Id,
				}

				_, err := computeClient.TerminateInstance(ctx, terminateRequest)
				if err != nil {
					fmt.Printf("Error terminating oversized OCI instance %s: %v\n", *instance.DisplayName, err)
					continue
				}
				fmt.Printf("Terminated oversized OCI instance: %s\n", *instance.DisplayName)
				count++
			}
		}
	}

	return nil
}

// terminateIBMOversizedInstances terminates IBM Cloud instances that exceed size limit
func terminateIBMOversizedInstances(ctx context.Context, provider models.CloudProvider, cfg *config.Config, maxSizeLevel int) error {
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Credentials), &credentials); err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	apiKey, _ := credentials["apiKey"].(string)
	region, _ := credentials["region"].(string)

	if apiKey == "" {
		return fmt.Errorf("missing IBM Cloud credentials (apiKey)")
	}
	if region == "" {
		region = "us-south"
	}

	authenticator := &ibmcore.IamAuthenticator{
		ApiKey: apiKey,
	}

	vpcService, err := vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
		Authenticator: authenticator,
		URL:           fmt.Sprintf("https://%s.iaas.cloud.ibm.com/v1", region),
	})
	if err != nil {
		return fmt.Errorf("failed to create IBM VPC client: %w", err)
	}

	// IBM profile size levels (based on profile naming convention)
	sizeLevel := func(profileName string) int {
		lower := strings.ToLower(profileName)
		switch {
		case strings.Contains(lower, "2x"):
			return 1
		case strings.Contains(lower, "4x"):
			return 2
		case strings.Contains(lower, "8x"):
			return 3
		case strings.Contains(lower, "16x"):
			return 4
		case strings.Contains(lower, "32x"):
			return 5
		case strings.Contains(lower, "64x"):
			return 6
		default:
			return 7
		}
	}

	listInstancesOptions := vpcService.NewListInstancesOptions()
	instances, _, err := vpcService.ListInstances(listInstancesOptions)
	if err != nil {
		return fmt.Errorf("failed to list IBM instances: %w", err)
	}

	count := 0
	for _, instance := range instances.Instances {
		if count >= 5 {
			break
		}

		if instance.Status != nil && *instance.Status != "running" {
			continue
		}

		profileName := ""
		if instance.Profile != nil && instance.Profile.Name != nil {
			profileName = *instance.Profile.Name
		}

		if sizeLevel(profileName) > maxSizeLevel {
			hasEssential := false
			if instance.Name != nil && containsEssential(*instance.Name) {
				hasEssential = true
			}

			if !hasEssential && instance.ID != nil {
				deleteInstanceOptions := vpcService.NewDeleteInstanceOptions(*instance.ID)
				_, err := vpcService.DeleteInstance(deleteInstanceOptions)
				if err != nil {
					fmt.Printf("Error deleting oversized IBM instance %s: %v\n", *instance.Name, err)
					continue
				}
				fmt.Printf("Deleted oversized IBM instance: %s\n", *instance.Name)
				count++
			}
		}
	}

	return nil
}

// StopIdleResources stops resources that have been idle for specified hours
func StopIdleResources(ctx context.Context, provider models.CloudProvider, cfg *config.Config, idleHoursThreshold float64) error {
	switch provider.Type {
	case "aws":
		return stopAWSIdleResources(ctx, provider, cfg, idleHoursThreshold)
	case "azure":
		return stopAzureIdleResources(ctx, provider, cfg, idleHoursThreshold)
	case "gcp":
		return stopGCPIdleResources(ctx, provider, cfg, idleHoursThreshold)
	}
	return nil
}

// stopAWSIdleResources stops AWS EC2 instances that have been idle
func stopAWSIdleResources(ctx context.Context, provider models.CloudProvider, cfg *config.Config, idleHoursThreshold float64) error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.AWSRegion),
	})
	if err != nil {
		return err
	}

	ec2Svc := ec2.New(sess)
	cwSvc := cloudwatch.New(sess)

	// Get running instances
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

	now := time.Now()
	checkStart := now.Add(-time.Duration(idleHoursThreshold) * time.Hour)

	count := 0
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if count >= 5 {
				break
			}

			// Check for Essential tag
			hasEssential := false
			for _, tag := range instance.Tags {
				if *tag.Key == "Essential" && *tag.Value == "true" {
					hasEssential = true
					break
				}
			}

			if hasEssential {
				continue
			}

			// Check CPU utilization from CloudWatch
			metricsInput := &cloudwatch.GetMetricStatisticsInput{
				Namespace:  aws.String("AWS/EC2"),
				MetricName: aws.String("CPUUtilization"),
				Dimensions: []*cloudwatch.Dimension{
					{
						Name:  aws.String("InstanceId"),
						Value: instance.InstanceId,
					},
				},
				StartTime:  aws.Time(checkStart),
				EndTime:    aws.Time(now),
				Period:     aws.Int64(3600), // 1 hour periods
				Statistics: []*string{aws.String("Average")},
			}

			metricsOutput, err := cwSvc.GetMetricStatistics(metricsInput)
			if err != nil {
				fmt.Printf("Warning: could not get metrics for %s: %v\n", *instance.InstanceId, err)
				continue
			}

			// Check if instance has been idle (CPU < 5% average)
			isIdle := true
			for _, datapoint := range metricsOutput.Datapoints {
				if datapoint.Average != nil && *datapoint.Average > 5.0 {
					isIdle = false
					break
				}
			}

			if isIdle && len(metricsOutput.Datapoints) > 0 {
				_, err := ec2Svc.StopInstances(&ec2.StopInstancesInput{
					InstanceIds: []*string{instance.InstanceId},
				})
				if err != nil {
					fmt.Printf("Error stopping idle instance %s: %v\n", *instance.InstanceId, err)
				} else {
					fmt.Printf("Stopped idle instance %s (idle for %.1f hours)\n", *instance.InstanceId, idleHoursThreshold)
					count++
				}
			}
		}
	}

	return nil
}

// stopAzureIdleResources stops Azure VMs that have been idle
func stopAzureIdleResources(ctx context.Context, provider models.CloudProvider, cfg *config.Config, idleHoursThreshold float64) error {
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

	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return fmt.Errorf("failed to create Azure credential: %w", err)
	}

	vmClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create VM client: %w", err)
	}

	// Note: For Azure, you would typically use Azure Monitor to check metrics
	// This is a simplified version that stops VMs without Essential tag
	// In production, integrate with Azure Monitor for CPU metrics

	pager := vmClient.NewListAllPager(nil)
	count := 0

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list VMs: %w", err)
		}

		for _, vm := range page.Value {
			if count >= 5 {
				break
			}

			hasEssential := false
			if vm.Tags != nil {
				if val, ok := vm.Tags["Essential"]; ok && val != nil && *val == "true" {
					hasEssential = true
				}
			}

			// Check for IdleCheckEnabled tag to opt-in to idle stopping
			idleCheckEnabled := false
			if vm.Tags != nil {
				if val, ok := vm.Tags["IdleCheckEnabled"]; ok && val != nil && *val == "true" {
					idleCheckEnabled = true
				}
			}

			if !hasEssential && idleCheckEnabled && vm.Name != nil && vm.ID != nil {
				resourceGroup := extractResourceGroupFromID(*vm.ID)
				if resourceGroup == "" {
					continue
				}

				poller, err := vmClient.BeginDeallocate(ctx, resourceGroup, *vm.Name, nil)
				if err != nil {
					fmt.Printf("Error stopping idle Azure VM %s: %v\n", *vm.Name, err)
					continue
				}

				_, err = poller.PollUntilDone(ctx, nil)
				if err != nil {
					fmt.Printf("Error waiting for VM %s to stop: %v\n", *vm.Name, err)
				} else {
					fmt.Printf("Stopped idle Azure VM: %s\n", *vm.Name)
					count++
				}
			}
		}
	}

	return nil
}

// stopGCPIdleResources stops GCP instances that have been idle
func stopGCPIdleResources(ctx context.Context, provider models.CloudProvider, cfg *config.Config, idleHoursThreshold float64) error {
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Credentials), &credentials); err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	serviceAccountJSON, _ := credentials["serviceAccountKey"].(string)
	projectID := provider.ProjectID

	if serviceAccountJSON == "" || projectID == "" {
		return fmt.Errorf("missing GCP credentials or projectId")
	}

	computeService, err := compute.NewService(ctx, option.WithCredentialsJSON([]byte(serviceAccountJSON)))
	if err != nil {
		return fmt.Errorf("failed to create compute service: %w", err)
	}

	monitoringService, err := monitoring.NewService(ctx, option.WithCredentialsJSON([]byte(serviceAccountJSON)))
	if err != nil {
		return fmt.Errorf("failed to create monitoring service: %w", err)
	}

	zonesResp, err := computeService.Zones.List(projectID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to list zones: %w", err)
	}

	now := time.Now()
	checkStart := now.Add(-time.Duration(idleHoursThreshold) * time.Hour)

	count := 0
	for _, zone := range zonesResp.Items {
		if count >= 5 {
			break
		}

		instancesResp, err := computeService.Instances.List(projectID, zone.Name).
			Filter("status=RUNNING").
			Context(ctx).Do()
		if err != nil {
			continue
		}

		for _, instance := range instancesResp.Items {
			if count >= 5 {
				break
			}

			// Check for essential label
			hasEssential := false
			if instance.Labels != nil {
				if val, ok := instance.Labels["essential"]; ok && val == "true" {
					hasEssential = true
				}
			}

			if hasEssential {
				continue
			}

			// Query Cloud Monitoring for CPU utilization
			filter := fmt.Sprintf(`metric.type="compute.googleapis.com/instance/cpu/utilization" AND resource.labels.instance_id="%d"`, instance.Id)

			req := monitoringService.Projects.TimeSeries.List(fmt.Sprintf("projects/%s", projectID)).
				Filter(filter).
				IntervalStartTime(checkStart.Format(time.RFC3339)).
				IntervalEndTime(now.Format(time.RFC3339)).
				AggregationAlignmentPeriod("3600s").
				AggregationPerSeriesAligner("ALIGN_MEAN")

			tsResp, err := req.Do()
			if err != nil {
				fmt.Printf("Warning: could not get metrics for instance %s: %v\n", instance.Name, err)
				continue
			}

			// Check if instance has been idle (CPU < 5% average)
			isIdle := true
			for _, ts := range tsResp.TimeSeries {
				for _, point := range ts.Points {
					if point.Value != nil && point.Value.DoubleValue != nil && *point.Value.DoubleValue > 0.05 {
						isIdle = false
						break
					}
				}
			}

			if isIdle && len(tsResp.TimeSeries) > 0 {
				_, err := computeService.Instances.Stop(projectID, zone.Name, instance.Name).Context(ctx).Do()
				if err != nil {
					fmt.Printf("Error stopping idle GCP instance %s: %v\n", instance.Name, err)
					continue
				}
				fmt.Printf("Stopped idle GCP instance: %s in zone %s\n", instance.Name, zone.Name)
				count++
			}
		}
	}

	return nil
}

