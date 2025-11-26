package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	models "finopsbridge/api/internal/models_"

	"github.com/gofiber/fiber/v2"
)

// GenerateRecommendations analyzes org's cloud spend and generates policy recommendations
func (h *Handlers) GenerateRecommendations(c *fiber.Ctx) error {
	orgID := c.Locals("orgId").(string)

	// Get all cloud providers for this org
	var providers []models.CloudProvider
	if err := h.DB.Where("organization_id = ?", orgID).Find(&providers).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch cloud providers",
		})
	}

	if len(providers) == 0 {
		return c.JSON([]interface{}{}) // No providers, no recommendations
	}

	// Get existing policies to avoid duplicate recommendations
	var existingPolicies []models.Policy
	h.DB.Where("organization_id = ?", orgID).Find(&existingPolicies)
	existingPolicyTypes := make(map[string]bool)
	for _, p := range existingPolicies {
		existingPolicyTypes[p.Type] = true
	}

	// Delete old pending recommendations
	h.DB.Where("organization_id = ? AND status = ?", orgID, "pending").Delete(&models.PolicyRecommendation{})

	// Analyze and generate recommendations
	recommendations := h.analyzeAndRecommend(orgID, providers, existingPolicyTypes)

	// Save recommendations to database
	for _, rec := range recommendations {
		h.DB.Create(&rec)
	}

	// Log activity
	h.logActivity(orgID, "recommendations_generated", fmt.Sprintf("Generated %d policy recommendations", len(recommendations)), nil)

	return c.JSON(recommendations)
}

// analyzeAndRecommend performs analysis and returns recommendations
func (h *Handlers) analyzeAndRecommend(orgID string, providers []models.CloudProvider, existingPolicyTypes map[string]bool) []models.PolicyRecommendation {
	var recommendations []models.PolicyRecommendation
	totalSpend := 0.0

	// Calculate total monthly spend
	for _, p := range providers {
		totalSpend += p.MonthlySpend
	}

	// Get all policy templates
	var templates []models.PolicyTemplate
	h.DB.Find(&templates)

	// Rule-based recommendation engine
	for _, template := range templates {
		// Skip if policy already exists
		if existingPolicyTypes[template.PolicyType] {
			continue
		}

		confidence, savings, reason, issues := h.evaluateTemplate(template, providers, totalSpend)

		if confidence > 0.3 { // Only recommend if confidence > 30%
			priority := "low"
			if confidence > 0.8 {
				priority = "critical"
			} else if confidence > 0.6 {
				priority = "high"
			} else if confidence > 0.4 {
				priority = "medium"
			}

			// Prepare suggested config based on analysis
			suggestedConfig := h.generateSuggestedConfig(template, providers, totalSpend)
			configJSON, _ := json.Marshal(suggestedConfig)

			// Prepare detected issues
			issuesJSON, _ := json.Marshal(issues)

			rec := models.PolicyRecommendation{
				OrganizationID:          orgID,
				PolicyTemplateID:        template.ID,
				Status:                  "pending",
				ConfidenceScore:         math.Round(confidence*100) / 100,
				EstimatedMonthlySavings: math.Round(savings*100) / 100,
				RecommendationReason:    reason,
				DetectedIssues:          string(issuesJSON),
				SuggestedConfig:         string(configJSON),
				Priority:                priority,
			}

			recommendations = append(recommendations, rec)
		}
	}

	return recommendations
}

// evaluateTemplate determines if a template is recommended
func (h *Handlers) evaluateTemplate(template models.PolicyTemplate, providers []models.CloudProvider, totalSpend float64) (float64, float64, string, []string) {
	var confidence float64
	var savings float64
	var reason string
	var issues []string

	switch template.PolicyType {
	case "max_spend":
		// Always recommend budget control if spend > $1000/month
		if totalSpend > 1000 {
			confidence = 0.95
			savings = totalSpend * 0.05 // 5% from awareness
			reason = fmt.Sprintf("Your organization spends $%.2f/month. A budget policy prevents unexpected overages and promotes cost awareness.", totalSpend)
			issues = []string{"No budget controls in place", "Risk of cost overruns"}
		}

	case "auto_stop_idle":
		// High confidence if multiple cloud providers (likely has dev/test resources)
		if len(providers) > 0 {
			confidence = 0.85
			savings = totalSpend * 0.15 // Estimated 15% savings from idle resources
			reason = "Idle resources are one of the top sources of cloud waste (typically 15-30% of total spend). This policy automatically stops resources with low CPU utilization."
			issues = []string{"Potentially running idle compute resources 24/7", "Dev/test resources not stopped after hours"}
		}

	case "require_tags":
		// Always recommend tagging for cost allocation
		confidence = 0.90
		savings = 0 // Indirect savings through better visibility
		reason = "Mandatory tagging enables cost allocation, chargeback, and identifying unowned resources. Essential for FinOps maturity."
		issues = []string{"Cannot track costs by team/project", "Difficulty identifying resource owners"}

	case "block_instance_type":
		// Recommend if high monthly spend (likely has oversized instances)
		if totalSpend > 5000 {
			confidence = 0.75
			savings = totalSpend * 0.10 // 10% from preventing oversized instances
			reason = "Prevent teams from deploying unnecessarily large instances. Organizations typically see 10-20% savings by rightsizing."
			issues = []string{"Risk of over-provisioning", "No guardrails on instance sizes"}
		}

	case "scheduled_start_stop":
		// Recommend for non-production environments
		if len(providers) > 0 {
			confidence = 0.80
			savings = totalSpend * 0.20 // 20% savings on dev/test
			reason = "Automatically start/stop dev and test environments during business hours. Typical savings: 50-70% on non-production workloads."
			issues = []string{"Non-production resources running 24/7", "Unnecessary weekend compute costs"}
		}

	case "unattached_cleanup":
		// Always recommend cleanup policies
		confidence = 0.70
		savings = totalSpend * 0.05 // 5% from cleanup
		reason = "Orphaned EBS volumes, snapshots, and unused Elastic IPs accumulate over time. Automatic cleanup prevents waste."
		issues = []string{"Storage costs increasing over time", "Unused resources accumulating"}

	case "rightsizing":
		// Recommend for organizations with significant spend
		if totalSpend > 3000 {
			confidence = 0.85
			savings = totalSpend * 0.25 // 25% from rightsizing
			reason = "Analyze actual CPU/memory utilization and recommend optimal instance sizes. Typical savings: 20-35% of compute costs."
			issues = []string{"Instances potentially oversized", "Paying for unused capacity"}
		}

	case "encryption_enforcement":
		// Always recommend for compliance
		confidence = 0.65
		savings = 0 // Compliance/security benefit
		reason = "Ensure all storage resources use encryption. Critical for SOC 2, HIPAA, and PCI-DSS compliance."
		issues = []string{"Potential compliance gaps", "Security risk from unencrypted data"}

	case "backup_enforcement":
		// Recommend for production environments
		if totalSpend > 2000 {
			confidence = 0.70
			savings = 0 // DR/compliance benefit
			reason = "Automate backup policies for critical databases and storage. Prevents data loss and ensures business continuity."
			issues = []string{"Inconsistent backup practices", "Data loss risk"}
		}

	case "reserved_instance":
		// Recommend if significant steady-state workload
		if totalSpend > 5000 {
			confidence = 0.80
			savings = totalSpend * 0.30 // 30% from RIs/Savings Plans
			reason = "Convert steady-state workloads to Reserved Instances or Savings Plans for 30-60% savings on compute."
			issues = []string{"High on-demand compute costs", "Missing commitment-based discounts"}
		}

	default:
		// Default low confidence for other templates
		confidence = 0.35
		savings = totalSpend * 0.05
		reason = "Consider this policy to improve cloud governance and cost optimization."
		issues = []string{"Additional optimization opportunity"}
	}

	return confidence, savings, reason, issues
}

// generateSuggestedConfig creates a suggested configuration for a template
func (h *Handlers) generateSuggestedConfig(template models.PolicyTemplate, providers []models.CloudProvider, totalSpend float64) map[string]interface{} {
	config := make(map[string]interface{})

	switch template.PolicyType {
	case "max_spend":
		// Suggest threshold as 110% of current spend
		config["threshold"] = math.Round(totalSpend * 1.1)
		config["currency"] = "USD"
		config["alertThresholds"] = []int{70, 85, 100}

	case "auto_stop_idle":
		config["idleHours"] = 24
		config["cpuThreshold"] = 5
		config["excludeTags"] = []string{"Essential:true"}

	case "require_tags":
		config["requiredTags"] = []string{"Owner", "Environment", "CostCenter", "Project"}
		config["enforcementLevel"] = "warning"

	case "block_instance_type":
		config["maxInstanceSize"] = map[string]string{
			"production":  "xlarge",
			"staging":     "large",
			"development": "medium",
		}

	case "scheduled_start_stop":
		config["schedule"] = map[string]interface{}{
			"timezone": "America/New_York",
			"weekdays": "08:00-18:00",
			"weekends": "off",
		}
		config["targetEnvironments"] = []string{"development", "staging", "test"}

	case "unattached_cleanup":
		config["retentionDays"] = map[string]int{
			"unattachedVolumes": 7,
			"unusedEIPs":        3,
			"oldSnapshots":      90,
		}

	case "rightsizing":
		config["utilizationThresholds"] = map[string]float64{
			"cpuDownsize": 0.25,
			"cpuUpsize":   0.80,
			"memoryDownsize": 0.30,
		}
		config["evaluationPeriod"] = 14

	case "reserved_instance":
		config["minUtilization"] = 0.75
		config["commitmentTerm"] = "1-year"
		config["paymentOption"] = "no-upfront"
	}

	return config
}

// ListRecommendations returns policy recommendations for an organization
func (h *Handlers) ListRecommendations(c *fiber.Ctx) error {
	orgID := c.Locals("orgId").(string)

	var recommendations []models.PolicyRecommendation
	if err := h.DB.Where("organization_id = ?", orgID).
		Order("priority DESC, confidence_score DESC").
		Find(&recommendations).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch recommendations",
		})
	}

	// Join with template data
	type RecommendationResponse struct {
		models.PolicyRecommendation
		Template models.PolicyTemplate `json:"template"`
	}

	var responses []RecommendationResponse
	for _, rec := range recommendations {
		var template models.PolicyTemplate
		h.DB.First(&template, "id = ?", rec.PolicyTemplateID)

		responses = append(responses, RecommendationResponse{
			PolicyRecommendation: rec,
			Template:             template,
		})
	}

	return c.JSON(responses)
}

// AcceptRecommendation marks a recommendation as accepted
func (h *Handlers) AcceptRecommendation(c *fiber.Ctx) error {
	recommendationID := c.Params("id")
	orgID := c.Locals("orgId").(string)

	var rec models.PolicyRecommendation
	if err := h.DB.Where("id = ? AND organization_id = ?", recommendationID, orgID).First(&rec).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Recommendation not found",
		})
	}

	rec.Status = "accepted"
	h.DB.Save(&rec)

	h.logActivity(orgID, "recommendation_accepted", "Accepted policy recommendation", nil)

	return c.JSON(rec)
}

// RejectRecommendation marks a recommendation as rejected
func (h *Handlers) RejectRecommendation(c *fiber.Ctx) error {
	recommendationID := c.Params("id")
	orgID := c.Locals("orgId").(string)

	type RejectRequest struct {
		Reason string `json:"reason"`
	}

	var req RejectRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var rec models.PolicyRecommendation
	if err := h.DB.Where("id = ? AND organization_id = ?", recommendationID, orgID).First(&rec).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Recommendation not found",
		})
	}

	now := time.Now()
	rec.Status = "rejected"
	rec.RejectedAt = &now
	rec.RejectionReason = req.Reason
	h.DB.Save(&rec)

	h.logActivity(orgID, "recommendation_rejected", "Rejected policy recommendation: "+req.Reason, nil)

	return c.JSON(rec)
}
