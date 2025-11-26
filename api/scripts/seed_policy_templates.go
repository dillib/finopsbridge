package main

import (
	"encoding/json"
	"fmt"
	"log"

	config "finopsbridge/api/internal/config_"
	database "finopsbridge/api/internal/database_"
	models "finopsbridge/api/internal/models_"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	fmt.Println("🌱 Seeding policy templates...")

	// Create categories
	categories := []models.PolicyCategory{
		{
			Name:        "Cost Control & Budget Management",
			Description: "Policies to control cloud spending and prevent budget overruns",
			Icon:        "💰",
			SortOrder:   1,
		},
		{
			Name:        "Resource Governance & Rightsizing",
			Description: "Optimize resource allocation and prevent over-provisioning",
			Icon:        "⚙️",
			SortOrder:   2,
		},
		{
			Name:        "Security & Compliance",
			Description: "Ensure security best practices and regulatory compliance",
			Icon:        "🔒",
			SortOrder:   3,
		},
		{
			Name:        "Operational Efficiency",
			Description: "Automate operations and improve system reliability",
			Icon:        "🚀",
			SortOrder:   4,
		},
		{
			Name:        "Data & Database Optimization",
			Description: "Optimize database costs and performance",
			Icon:        "💾",
			SortOrder:   5,
		},
	}

	for i := range categories {
		db.FirstOrCreate(&categories[i], models.PolicyCategory{Name: categories[i].Name})
	}

	fmt.Println("✅ Created 5 policy categories")

	// Helper function to create JSON strings
	toJSON := func(v interface{}) string {
		b, _ := json.Marshal(v)
		return string(b)
	}

	// Create policy templates
	templates := []models.PolicyTemplate{
		// COST CONTROL
		{
			CategoryID:      categories[0].ID,
			Name:            "Maximum Monthly Spend",
			Description:     "Prevent monthly cloud spending from exceeding defined budget limits. Get alerts at 70%, 85%, and 100% of budget.",
			PolicyType:      "max_spend",
			EstimatedSavings: "5-15% cost reduction through awareness",
			Difficulty:      "easy",
			CloudProviders:  toJSON([]string{"aws", "azure", "gcp", "oci", "ibm"}),
			BusinessImpact:  "Prevents unexpected cost overruns and promotes cost awareness across teams. Essential for budget planning and financial control.",
			DefaultConfig: toJSON(map[string]interface{}{
				"threshold": 10000,
				"currency":  "USD",
				"alertThresholds": []int{70, 85, 100},
			}),
			Tags: toJSON([]string{"budget", "alerts", "cost-control"}),
			RequiredPermissions: toJSON([]string{"billing:read"}),
			ComplianceFrameworks: toJSON([]string{"finops"}),
			RegoTemplate: `package finopsbridge.policies.max_spend

default allow = false

allow {
	input.monthlySpend < data.policy.config.threshold
}

violation[msg] {
	input.monthlySpend >= data.policy.config.threshold
	msg := sprintf("Monthly spend $%.2f exceeds threshold $%.2f", [input.monthlySpend, data.policy.config.threshold])
}`,
		},
		{
			CategoryID:      categories[0].ID,
			Name:            "Daily Spend Anomaly Detection",
			Description:     "Detect unusual spending patterns using AI-based anomaly detection. Alerts when daily spend exceeds 150% of 7-day average.",
			PolicyType:      "anomaly_detection",
			EstimatedSavings: "10-20% by catching waste early",
			Difficulty:      "medium",
			CloudProviders:  toJSON([]string{"aws", "azure", "gcp"}),
			BusinessImpact:  "Early detection prevents 40-60% of cost waste incidents by identifying spikes before they become major issues.",
			DefaultConfig: toJSON(map[string]interface{}{
				"dailyThreshold": 1.5,
				"weeklyBaseline": 7,
			}),
			Tags: toJSON([]string{"anomaly", "ai", "monitoring"}),
			RequiredPermissions: toJSON([]string{"billing:read", "cloudwatch:read"}),
			ComplianceFrameworks: toJSON([]string{"finops"}),
			RegoTemplate: `package finopsbridge.policies.anomaly

default allow = true

violation[msg] {
	input.dailySpend > (input.averageSpend * data.policy.config.dailyThreshold)
	msg := sprintf("Daily spend anomaly: $%.2f (%.0f%% above baseline)", [input.dailySpend, ((input.dailySpend / input.averageSpend - 1) * 100)])
}`,
		},
		{
			CategoryID:      categories[0].ID,
			Name:            "Reserved Instance Optimization",
			Description:     "Ensure optimal use of Reserved Instances and Savings Plans. Identifies steady-state workloads running on expensive on-demand pricing.",
			PolicyType:      "reserved_instance",
			EstimatedSavings: "30-60% on compute costs",
			Difficulty:      "hard",
			CloudProviders:  toJSON([]string{"aws", "azure", "gcp"}),
			BusinessImpact:  "Up to 60% savings on compute costs through commitment-based pricing. Typical ROI: 45-55% annually.",
			DefaultConfig: toJSON(map[string]interface{}{
				"minUtilization": 0.75,
				"maxOnDemandPercentage": 0.40,
				"commitmentTerm": "1-year",
			}),
			Tags: toJSON([]string{"reserved-instances", "savings-plans", "commitments"}),
			RequiredPermissions: toJSON([]string{"ec2:describe", "ce:get"}),
			ComplianceFrameworks: toJSON([]string{"finops"}),
			RegoTemplate: `package finopsbridge.policies.reserved_instance

violation[msg] {
	input.riUtilization < data.policy.config.minUtilization
	msg := sprintf("RI utilization %.0f%% below target %.0f%%", [input.riUtilization * 100, data.policy.config.minUtilization * 100])
}`,
		},

		// RESOURCE GOVERNANCE
		{
			CategoryID:      categories[1].ID,
			Name:            "Block Oversized Instances",
			Description:     "Prevent deployment of instances larger than necessary based on environment (production, staging, development).",
			PolicyType:      "block_instance_type",
			EstimatedSavings: "15-30% compute cost reduction",
			Difficulty:      "easy",
			CloudProviders:  toJSON([]string{"aws", "azure", "gcp", "oci", "ibm"}),
			BusinessImpact:  "Prevents over-provisioning and enforces cost-conscious instance selection. Typical savings: 15-30% on compute.",
			DefaultConfig: toJSON(map[string]interface{}{
				"maxInstanceSize": map[string]string{
					"production":  "xlarge",
					"staging":     "large",
					"development": "medium",
				},
			}),
			Tags: toJSON([]string{"governance", "instance-size", "guardrails"}),
			RequiredPermissions: toJSON([]string{"ec2:describe"}),
			ComplianceFrameworks: toJSON([]string{"finops", "ccoe"}),
			RegoTemplate: `package finopsbridge.policies.block_instance

violation[msg] {
	blocked := data.policy.config.blockedTypes[_]
	input.instanceType == blocked
	msg := sprintf("Instance type %s is not allowed", [input.instanceType])
}`,
		},
		{
			CategoryID:      categories[1].ID,
			Name:            "Auto-Stop Idle Resources",
			Description:     "Automatically stop resources with low CPU utilization (<5%) for configurable hours. Prevents waste from forgotten resources.",
			PolicyType:      "auto_stop_idle",
			EstimatedSavings: "$15K-50K/month for mid-sized orgs",
			Difficulty:      "medium",
			CloudProviders:  toJSON([]string{"aws", "azure", "gcp"}),
			BusinessImpact:  "Eliminates 66% of idle resource waste. Typical savings: 15-30% of total compute costs.",
			DefaultConfig: toJSON(map[string]interface{}{
				"idleHours": 24,
				"cpuThreshold": 5,
				"excludeTags": []string{"Essential:true", "AlwaysOn:true"},
			}),
			Tags: toJSON([]string{"idle", "automation", "waste-reduction"}),
			RequiredPermissions: toJSON([]string{"ec2:stop", "cloudwatch:get"}),
			ComplianceFrameworks: toJSON([]string{"finops"}),
			RegoTemplate: `package finopsbridge.policies.auto_stop_idle

violation[msg] {
	input.cpuUtilization < data.policy.config.cpuThreshold
	input.idleHours >= data.policy.config.idleHours
	msg := sprintf("Resource idle for %d hours with %.1f%% CPU", [input.idleHours, input.cpuUtilization])
}`,
		},
		{
			CategoryID:      categories[1].ID,
			Name:            "Scheduled Start/Stop",
			Description:     "Automatically start and stop dev/test environments during business hours. Massive savings on non-production workloads.",
			PolicyType:      "scheduled_start_stop",
			EstimatedSavings: "50-70% on non-production",
			Difficulty:      "medium",
			CloudProviders:  toJSON([]string{"aws", "azure", "gcp"}),
			BusinessImpact:  "65% savings on non-production environments. Typical savings: $5K-20K/month per environment.",
			DefaultConfig: toJSON(map[string]interface{}{
				"schedule": map[string]interface{}{
					"timezone": "America/New_York",
					"weekdays": "08:00-18:00",
					"weekends": "off",
				},
				"targetEnvironments": []string{"development", "staging", "test"},
			}),
			Tags: toJSON([]string{"scheduling", "dev-test", "automation"}),
			RequiredPermissions: toJSON([]string{"ec2:start", "ec2:stop"}),
			ComplianceFrameworks: toJSON([]string{"finops"}),
			RegoTemplate: `package finopsbridge.policies.scheduled_start_stop

violation[msg] {
	not is_business_hours
	input.status == "running"
	input.environment != "production"
	msg := sprintf("Non-production resource running outside business hours (%s)", [input.environment])
}`,
		},
		{
			CategoryID:      categories[1].ID,
			Name:            "Unattached Resource Cleanup",
			Description:     "Identify and remove unused cloud resources: unattached EBS volumes, unused Elastic IPs, empty load balancers, old snapshots.",
			PolicyType:      "unattached_cleanup",
			EstimatedSavings: "10-15% storage cost reduction",
			Difficulty:      "easy",
			CloudProviders:  toJSON([]string{"aws", "azure", "gcp"}),
			BusinessImpact:  "Prevents accumulation of orphaned resources. Typical savings: 10-15% on storage costs.",
			DefaultConfig: toJSON(map[string]interface{}{
				"retentionDays": map[string]int{
					"unattachedVolumes": 7,
					"unusedEIPs":        3,
					"oldSnapshots":      90,
				},
			}),
			Tags: toJSON([]string{"cleanup", "storage", "waste-reduction"}),
			RequiredPermissions: toJSON([]string{"ec2:delete", "ec2:describe"}),
			ComplianceFrameworks: toJSON([]string{"finops"}),
			RegoTemplate: `package finopsbridge.policies.unattached_cleanup

violation[msg] {
	input.state == "available"
	input.daysUnattached >= data.policy.config.retentionDays.unattachedVolumes
	msg := sprintf("Volume unattached for %d days", [input.daysUnattached])
}`,
		},
		{
			CategoryID:      categories[1].ID,
			Name:            "Performance-Based Rightsizing",
			Description:     "Analyze actual CPU and memory utilization to recommend optimal instance sizes. Downsize underutilized, upsize overloaded.",
			PolicyType:      "rightsizing",
			EstimatedSavings: "25-35% compute cost reduction",
			Difficulty:      "hard",
			CloudProviders:  toJSON([]string{"aws", "azure", "gcp"}),
			BusinessImpact:  "25-35% compute cost reduction while maintaining or improving performance.",
			DefaultConfig: toJSON(map[string]interface{}{
				"utilizationThresholds": map[string]float64{
					"cpuDownsize":    0.25,
					"cpuUpsize":      0.80,
					"memoryDownsize": 0.30,
				},
				"evaluationPeriod": 14,
			}),
			Tags: toJSON([]string{"rightsizing", "optimization", "performance"}),
			RequiredPermissions: toJSON([]string{"ec2:describe", "cloudwatch:get"}),
			ComplianceFrameworks: toJSON([]string{"finops"}),
			RegoTemplate: `package finopsbridge.policies.rightsizing

violation[msg] {
	input.cpuUtilization < data.policy.config.utilizationThresholds.cpuDownsize
	input.evaluationDays >= data.policy.config.evaluationPeriod
	msg := sprintf("Downsize recommended: CPU %.1f%% for %d days", [input.cpuUtilization * 100, input.evaluationDays])
}`,
		},

		// SECURITY & COMPLIANCE
		{
			CategoryID:      categories[2].ID,
			Name:            "Mandatory Tagging",
			Description:     "Enforce required tags on all cloud resources for cost allocation, ownership tracking, and compliance.",
			PolicyType:      "require_tags",
			EstimatedSavings: "Indirect through visibility",
			Difficulty:      "easy",
			CloudProviders:  toJSON([]string{"aws", "azure", "gcp", "oci", "ibm"}),
			BusinessImpact:  "95% tag compliance enables accurate cost allocation and chargeback. Essential for FinOps maturity.",
			DefaultConfig: toJSON(map[string]interface{}{
				"requiredTags": []string{"Owner", "Environment", "CostCenter", "Project"},
				"enforcementLevel": "hard",
			}),
			Tags: toJSON([]string{"tagging", "cost-allocation", "governance"}),
			RequiredPermissions: toJSON([]string{"tag:describe"}),
			ComplianceFrameworks: toJSON([]string{"finops", "itil"}),
			RegoTemplate: `package finopsbridge.policies.require_tags

violation[msg] {
	required := data.policy.config.requiredTags[_]
	not input.tags[required]
	msg := sprintf("Missing required tag: %s", [required])
}`,
		},
		{
			CategoryID:      categories[2].ID,
			Name:            "Encryption Enforcement",
			Description:     "Ensure all storage resources use encryption at rest. Critical for SOC 2, HIPAA, and PCI-DSS compliance.",
			PolicyType:      "encryption_enforcement",
			EstimatedSavings: "Compliance/security benefit",
			Difficulty:      "medium",
			CloudProviders:  toJSON([]string{"aws", "azure", "gcp"}),
			BusinessImpact:  "Ensures SOC 2, HIPAA, PCI-DSS compliance. Prevents data breaches and regulatory fines.",
			DefaultConfig: toJSON(map[string]interface{}{
				"encryptionRequired": true,
				"keyManagement": "customer-managed",
				"resources": []string{"s3", "ebs", "rds", "efs"},
			}),
			Tags: toJSON([]string{"encryption", "security", "compliance"}),
			RequiredPermissions: toJSON([]string{"kms:describe", "s3:describe"}),
			ComplianceFrameworks: toJSON([]string{"soc2", "hipaa", "pci-dss"}),
			RegoTemplate: `package finopsbridge.policies.encryption

violation[msg] {
	not input.encrypted
	msg := sprintf("%s resource is not encrypted", [input.resourceType])
}`,
		},
		{
			CategoryID:      categories[2].ID,
			Name:            "Public Access Prevention",
			Description:     "Prevent accidental public exposure of S3 buckets, databases, and other sensitive resources.",
			PolicyType:      "public_access_prevention",
			EstimatedSavings: "Risk mitigation",
			Difficulty:      "medium",
			CloudProviders:  toJSON([]string{"aws", "azure", "gcp"}),
			BusinessImpact:  "Prevents 90% of accidental data exposure incidents. Critical for security and compliance.",
			DefaultConfig: toJSON(map[string]interface{}{
				"blockedPorts": []int{22, 3389, 3306, 5432},
				"allowedPublicResources": []string{"cloudfront", "alb-waf"},
			}),
			Tags: toJSON([]string{"security", "data-protection", "compliance"}),
			RequiredPermissions: toJSON([]string{"s3:getBucketPolicy", "ec2:describeSecurityGroups"}),
			ComplianceFrameworks: toJSON([]string{"soc2", "hipaa", "iso27001"}),
			RegoTemplate: `package finopsbridge.policies.public_access

violation[msg] {
	input.publicAccess == true
	msg := sprintf("%s has public access enabled", [input.resourceId])
}`,
		},

		// OPERATIONAL EFFICIENCY
		{
			CategoryID:      categories[3].ID,
			Name:            "Backup and Disaster Recovery",
			Description:     "Automate backup policies for critical databases and ensure business continuity with tested DR procedures.",
			PolicyType:      "backup_enforcement",
			EstimatedSavings: "DR/compliance benefit",
			Difficulty:      "medium",
			CloudProviders:  toJSON([]string{"aws", "azure", "gcp"}),
			BusinessImpact:  "99.9% data durability. Ensures RTO < 4 hours, RPO < 1 hour for critical systems.",
			DefaultConfig: toJSON(map[string]interface{}{
				"backupRetention": map[string]int{
					"production": 30,
					"staging":    7,
					"development": 3,
				},
				"requireCrossRegion": true,
			}),
			Tags: toJSON([]string{"backup", "disaster-recovery", "compliance"}),
			RequiredPermissions: toJSON([]string{"backup:describe", "rds:describe"}),
			ComplianceFrameworks: toJSON([]string{"soc2", "iso27001"}),
			RegoTemplate: `package finopsbridge.policies.backup

violation[msg] {
	not input.backupEnabled
	input.environment == "production"
	msg := "Production database does not have automated backups enabled"
}`,
		},
		{
			CategoryID:      categories[3].ID,
			Name:            "Storage Lifecycle Management",
			Description:     "Automatically tier S3 data to lower-cost storage classes based on access patterns. Glacier for archival data.",
			PolicyType:      "lifecycle_management",
			EstimatedSavings: "50-70% storage cost reduction",
			Difficulty:      "easy",
			CloudProviders:  toJSON([]string{"aws", "azure", "gcp"}),
			BusinessImpact:  "50-70% storage cost reduction through intelligent tiering. Automatic compliance with retention policies.",
			DefaultConfig: toJSON(map[string]interface{}{
				"s3Lifecycle": map[string]int{
					"standardToIA": 30,
					"iaToGlacier": 90,
					"deleteAfter": 365,
				},
			}),
			Tags: toJSON([]string{"storage", "lifecycle", "cost-optimization"}),
			RequiredPermissions: toJSON([]string{"s3:putLifecycleConfiguration"}),
			ComplianceFrameworks: toJSON([]string{"finops"}),
			RegoTemplate: `package finopsbridge.policies.lifecycle

violation[msg] {
	not input.lifecyclePolicyEnabled
	input.bucketSize > 1000000000
	msg := "Large S3 bucket without lifecycle policy"
}`,
		},

		// DATABASE OPTIMIZATION
		{
			CategoryID:      categories[4].ID,
			Name:            "Database Rightsizing",
			Description:     "Ensure databases are properly sized based on actual CPU, memory, and connection utilization patterns.",
			PolicyType:      "database_rightsizing",
			EstimatedSavings: "30-40% database cost savings",
			Difficulty:      "hard",
			CloudProviders:  toJSON([]string{"aws", "azure", "gcp"}),
			BusinessImpact:  "30-40% database cost savings while maintaining performance SLAs.",
			DefaultConfig: toJSON(map[string]interface{}{
				"cpuThreshold": 0.20,
				"storageThreshold": 0.80,
				"evaluationPeriod": 14,
			}),
			Tags: toJSON([]string{"database", "rightsizing", "rds"}),
			RequiredPermissions: toJSON([]string{"rds:describe", "cloudwatch:get"}),
			ComplianceFrameworks: toJSON([]string{"finops"}),
			RegoTemplate: `package finopsbridge.policies.database_rightsizing

violation[msg] {
	input.cpuUtilization < data.policy.config.cpuThreshold
	input.evaluationDays >= data.policy.config.evaluationPeriod
	msg := sprintf("Database underutilized: %.1f%% CPU for %d days", [input.cpuUtilization * 100, input.evaluationDays])
}`,
		},
	}

	for i := range templates {
		db.FirstOrCreate(&templates[i], models.PolicyTemplate{Name: templates[i].Name})
	}

	fmt.Printf("✅ Created %d policy templates\n", len(templates))
	fmt.Println("🎉 Policy template seeding complete!")
}
