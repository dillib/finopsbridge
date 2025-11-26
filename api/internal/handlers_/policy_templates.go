package handlers

import (
	"encoding/json"

	models "finopsbridge/api/internal/models_"

	"github.com/gofiber/fiber/v2"
)

// ListPolicyCategories returns all policy categories with templates
func (h *Handlers) ListPolicyCategories(c *fiber.Ctx) error {
	var categories []models.PolicyCategory

	if err := h.DB.Preload("Templates").Order("sort_order ASC").Find(&categories).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch policy categories",
		})
	}

	return c.JSON(categories)
}

// ListPolicyTemplates returns all policy templates
func (h *Handlers) ListPolicyTemplates(c *fiber.Ctx) error {
	var templates []models.PolicyTemplate

	// Optional filters
	categoryID := c.Query("category")
	cloudProvider := c.Query("cloud_provider")
	difficulty := c.Query("difficulty")

	query := h.DB.Model(&models.PolicyTemplate{})

	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}

	if cloudProvider != "" {
		query = query.Where("cloud_providers LIKE ?", "%\""+cloudProvider+"\"%")
	}

	if difficulty != "" {
		query = query.Where("difficulty = ?", difficulty)
	}

	// Order by popularity (usage count)
	if err := query.Order("usage_count DESC").Find(&templates).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch policy templates",
		})
	}

	return c.JSON(templates)
}

// GetPolicyTemplate returns a single policy template by ID
func (h *Handlers) GetPolicyTemplate(c *fiber.Ctx) error {
	templateID := c.Params("id")

	var template models.PolicyTemplate
	if err := h.DB.First(&template, "id = ?", templateID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Policy template not found",
		})
	}

	return c.JSON(template)
}

// DeployPolicyTemplate creates a policy from a template
func (h *Handlers) DeployPolicyTemplate(c *fiber.Ctx) error {
	templateID := c.Params("id")
	orgID := c.Locals("orgId").(string)

	// Get the template
	var template models.PolicyTemplate
	if err := h.DB.First(&template, "id = ?", templateID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Policy template not found",
		})
	}

	// Parse request body for custom configuration
	type DeployRequest struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Config      map[string]interface{} `json:"config"`
	}

	var req DeployRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Merge custom config with default config
	configJSON, err := mergeConfigs(template.DefaultConfig, req.Config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to merge configurations",
		})
	}

	// Create new policy from template
	policy := models.Policy{
		OrganizationID: orgID,
		Name:           req.Name,
		Description:    req.Description,
		Type:           template.PolicyType,
		Enabled:        true,
		Rego:           template.RegoTemplate,
		Config:         configJSON,
	}

	if err := h.DB.Create(&policy).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create policy",
		})
	}

	// Increment template usage count
	h.DB.Model(&template).Update("usage_count", template.UsageCount+1)

	// Log activity
	h.logActivity(orgID, "policy_created", "Created policy from template: "+template.Name, nil)

	return c.Status(201).JSON(policy)
}

// Helper function to merge configurations
func mergeConfigs(defaultConfigJSON string, customConfig map[string]interface{}) (string, error) {
	// Parse default config
	var defaultConfig map[string]interface{}
	if defaultConfigJSON != "" {
		if err := json.Unmarshal([]byte(defaultConfigJSON), &defaultConfig); err != nil {
			return "", err
		}
	} else {
		defaultConfig = make(map[string]interface{})
	}

	// Merge with custom config (custom config overrides defaults)
	for key, value := range customConfig {
		defaultConfig[key] = value
	}

	// Convert back to JSON
	configBytes, err := json.Marshal(defaultConfig)
	if err != nil {
		return "", err
	}

	return string(configBytes), nil
}
