package handlers

import (
	models "finopsbridge/api/internal/models_"

	"github.com/gofiber/fiber/v2"
)

// SeedDatabase seeds the database with initial data (categories and policy templates)
func (h *Handlers) SeedDatabase(c *fiber.Ctx) error {
	// Check if already seeded
	var count int64
	h.DB.Model(&models.PolicyCategory{}).Count(&count)
	if count > 0 {
		return c.JSON(fiber.Map{
			"message": "Database already seeded",
			"categories_count": count,
		})
	}

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
			Icon:        "🗄️",
			SortOrder:   5,
		},
	}

	for i := range categories {
		if err := h.DB.Create(&categories[i]).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to create categories: " + err.Error(),
			})
		}
	}

	// Now create policy templates for each category
	templates := h.getPolicyTemplates(categories)

	for i := range templates {
		if err := h.DB.Create(&templates[i]).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to create template: " + err.Error(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"message": "Database seeded successfully",
		"categories": len(categories),
		"templates": len(templates),
	})
}

func (h *Handlers) getPolicyTemplates(categories []models.PolicyCategory) []models.PolicyTemplate {
	costCategoryID := categories[0].ID
	resourceCategoryID := categories[1].ID
	securityCategoryID := categories[2].ID
	operationalCategoryID := categories[3].ID

	templates := []models.PolicyTemplate{
		{
			CategoryID:          costCategoryID,
			Name:                "Monthly Spend Limit",
			Description:         "Prevent cloud spending from exceeding monthly budgets",
			PolicyType:          "max_spend",
			DefaultConfig:       `{"max_monthly_spend": 10000}`,
			RegoTemplate:        `package finops\n\ndefault allow = false\n\nallow {\n    input.monthly_spend < input.config.max_monthly_spend\n}`,
			EstimatedSavings:    "20-30% reduction in unexpected costs",
			Difficulty:          "easy",
			RequiredPermissions: `["billing:read", "budget:write"]`,
			Tags:                `["cost-control", "budget", "spending-limit"]`,
			CloudProviders:      `["aws", "azure", "gcp"]`,
			ComplianceFrameworks: `["FinOps"]`,
			BusinessImpact:      "Prevents budget overruns and ensures predictable cloud costs",
			UsageCount:          0,
		},
		{
			CategoryID:          resourceCategoryID,
			Name:                "Block Expensive Instance Types",
			Description:         "Prevent deployment of unnecessarily large instance types",
			PolicyType:          "block_instance_type",
			DefaultConfig:       `{"blocked_instance_types": ["*.24xlarge", "*.32xlarge"]}`,
			RegoTemplate:        `package finops\n\ndefault allow = true\n\nallow = false {\n    some pattern\n    input.config.blocked_instance_types[pattern]\n    glob.match(pattern, [], input.instance_type)\n}`,
			EstimatedSavings:    "40-60% on compute costs",
			Difficulty:          "easy",
			RequiredPermissions: `["compute:read", "policy:write"]`,
			Tags:                `["rightsizing", "compute", "instance-type"]`,
			CloudProviders:      `["aws", "azure", "gcp"]`,
			ComplianceFrameworks: `["FinOps"]`,
			BusinessImpact:      "Prevents over-provisioning and reduces compute waste",
			UsageCount:          0,
		},
		{
			CategoryID:          operationalCategoryID,
			Name:                "Auto-Stop Idle Resources",
			Description:         "Automatically stop resources that are idle for extended periods",
			PolicyType:          "auto_stop_idle",
			DefaultConfig:       `{"idle_threshold_hours": 24, "cpu_threshold_percent": 5}`,
			RegoTemplate:        `package finops\n\ndefault allow = true\n\nviolation[msg] {\n    input.idle_hours > input.config.idle_threshold_hours\n    input.cpu_utilization < input.config.cpu_threshold_percent\n    msg := sprintf("Resource %s has been idle for %d hours", [input.resource_id, input.idle_hours])\n}`,
			EstimatedSavings:    "30-50% on idle resource costs",
			Difficulty:          "medium",
			RequiredPermissions: `["compute:read", "compute:stop", "monitoring:read"]`,
			Tags:                `["automation", "idle-detection", "cost-optimization"]`,
			CloudProviders:      `["aws", "azure", "gcp"]`,
			ComplianceFrameworks: `["FinOps"]`,
			BusinessImpact:      "Eliminates waste from forgotten or unused resources",
			UsageCount:          0,
		},
		{
			CategoryID:          securityCategoryID,
			Name:                "Require Resource Tags",
			Description:         "Enforce tagging standards for cost allocation and governance",
			PolicyType:          "require_tags",
			DefaultConfig:       `{"required_tags": ["Environment", "Owner", "CostCenter", "Project"]}`,
			RegoTemplate:        `package finops\n\ndefault allow = false\n\nallow {\n    required_tags := input.config.required_tags\n    count([tag | tag := required_tags[_]; input.tags[tag]]) == count(required_tags)\n}`,
			EstimatedSavings:    "10-15% through better cost visibility",
			Difficulty:          "easy",
			RequiredPermissions: `["tags:read", "policy:write"]`,
			Tags:                `["governance", "tagging", "compliance"]`,
			CloudProviders:      `["aws", "azure", "gcp"]`,
			ComplianceFrameworks: `["FinOps", "CIS"]`,
			BusinessImpact:      "Enables accurate cost allocation and chargeback",
			UsageCount:          0,
		},
	}

	return templates
}
