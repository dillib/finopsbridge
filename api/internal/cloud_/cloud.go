package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"finopsbridge/api/internal/config_"
	"finopsbridge/api/internal/models_"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func FetchAWSBilling(ctx context.Context, provider models_.CloudProvider, cfg *config_.Config) (map[string]interface{}, error) {
	var credentials map[string]interface{}
	json.Unmarshal([]byte(provider.Credentials), &credentials)

	roleArn, ok := credentials["roleArn"].(string)
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
		if amount, ok := result.ResultsByTime[0].Total["BlendedCost"].Amount; ok {
			fmt.Sscanf(*amount, "%f", &monthlySpend)
		}
	}

	return map[string]interface{}{
		"monthlySpend": monthlySpend,
		"currency":      "USD",
	}, nil
}

func FetchAzureBilling(ctx context.Context, provider models_.CloudProvider, cfg *config_.Config) (map[string]interface{}, error) {
	// Azure billing implementation
	// This would use Azure Cost Management API
	return map[string]interface{}{
		"monthlySpend": 0.0,
		"currency":     "USD",
	}, nil
}

func FetchGCPBilling(ctx context.Context, provider models_.CloudProvider, cfg *config_.Config) (map[string]interface{}, error) {
	// GCP billing implementation
	// This would use GCP Billing API
	return map[string]interface{}{
		"monthlySpend": 0.0,
		"currency":     "USD",
	}, nil
}

func StopNonEssentialResources(ctx context.Context, provider models_.CloudProvider, cfg *config_.Config) error {
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

func stopAWSNonEssentialResources(ctx context.Context, provider models_.CloudProvider, cfg *config_.Config) error {
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

func stopAzureNonEssentialResources(ctx context.Context, provider models_.CloudProvider, cfg *config_.Config) error {
	// Azure implementation
	return nil
}

func stopGCPNonEssentialResources(ctx context.Context, provider models_.CloudProvider, cfg *config_.Config) error {
	// GCP implementation
	return nil
}

func TerminateOversizedInstances(ctx context.Context, provider models_.CloudProvider, cfg *config_.Config) error {
	// Similar to StopNonEssentialResources but terminates instead
	return nil
}

func StopIdleResources(ctx context.Context, provider models_.CloudProvider, cfg *config_.Config) error {
	// Stop resources that have been idle for specified hours
	return nil
}

