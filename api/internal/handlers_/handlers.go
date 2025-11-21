package handlers

import (
	"encoding/json"
	"finopsbridge/api/internal/config_"
	"finopsbridge/api/internal/middleware_"
	"finopsbridge/api/internal/models_"
	"finopsbridge/api/internal/opa_"
	"finopsbridge/api/internal/policygen_"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Handlers struct {
	DB       *gorm.DB
	OPA      *opa_.Engine
	Config   *config_.Config
}

func New(db *gorm.DB, opaEngine *opa_.Engine, cfg *config_.Config) *Handlers {
	return &Handlers{
		DB:     db,
		OPA:    opaEngine,
		Config: cfg,
	}
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal server error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error": message,
	})
}

func (h *Handlers) CreateWaitlistEntry(c *fiber.Ctx) error {
	var req struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Company string `json:"company"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	entry := models_.WaitlistEntry{
		Email:   req.Email,
		Name:    req.Name,
		Company: req.Company,
	}

	if err := h.DB.Create(&entry).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create waitlist entry",
		})
	}

	return c.JSON(entry)
}

func (h *Handlers) GetDashboardStats(c *fiber.Ctx) error {
	orgID := middleware_.GetOrgID(c)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Organization ID required",
		})
	}

	// Get total spend
	var totalSpend float64
	h.DB.Model(&models_.CloudProvider{}).
		Where("organization_id = ? AND status = ?", orgID, "connected").
		Select("COALESCE(SUM(monthly_spend), 0)").
		Scan(&totalSpend)

	// Get active policies count
	var activePolicies int64
	h.DB.Model(&models_.Policy{}).
		Where("organization_id = ? AND enabled = ?", orgID, true).
		Count(&activePolicies)

	// Get connected clouds count
	var connectedClouds int64
	h.DB.Model(&models_.CloudProvider{}).
		Where("organization_id = ? AND status = ?", orgID, "connected").
		Count(&connectedClouds)

	// Get violations count (this month)
	var violations int64
	startOfMonth := time.Now().AddDate(0, 0, -time.Now().Day()+1)
	h.DB.Model(&models_.PolicyViolation{}).
		Joins("JOIN policies ON policy_violations.policy_id = policies.id").
		Where("policies.organization_id = ? AND policy_violations.created_at >= ?", orgID, startOfMonth).
		Count(&violations)

	// Get spend by provider
	var spendByProvider []struct {
		Provider string  `json:"provider"`
		Amount   float64 `json:"amount"`
	}
	h.DB.Model(&models_.CloudProvider{}).
		Where("organization_id = ? AND status = ?", orgID, "connected").
		Select("type as provider, COALESCE(SUM(monthly_spend), 0) as amount").
		Group("type").
		Scan(&spendByProvider)

	// Get spend trend (last 6 months)
	var spendTrend []struct {
		Date   string  `json:"date"`
		Amount float64 `json:"amount"`
	}
	for i := 5; i >= 0; i-- {
		month := time.Now().AddDate(0, -i, 0)
		var amount float64
		h.DB.Model(&models_.CloudProvider{}).
			Where("organization_id = ? AND status = ?", orgID, "connected").
			Select("COALESCE(SUM(monthly_spend), 0)").
			Scan(&amount)
		spendTrend = append(spendTrend, struct {
			Date   string  `json:"date"`
			Amount float64 `json:"amount"`
		}{
			Date:   month.Format("2006-01-02"),
			Amount: amount,
		})
	}

	// Get remediations count
	var remediations int64
	h.DB.Model(&models_.PolicyViolation{}).
		Joins("JOIN policies ON policy_violations.policy_id = policies.id").
		Where("policies.organization_id = ? AND policy_violations.status = ?", orgID, "remediated").
		Count(&remediations)

	return c.JSON(fiber.Map{
		"totalSpend":       totalSpend,
		"activePolicies":   activePolicies,
		"connectedClouds": connectedClouds,
		"violations":       violations,
		"remediations":     remediations,
		"spendByProvider":  spendByProvider,
		"spendTrend":       spendTrend,
	})
}

func (h *Handlers) ListPolicies(c *fiber.Ctx) error {
	orgID := middleware_.GetOrgID(c)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Organization ID required",
		})
	}

	var policies []models_.Policy
	if err := h.DB.Where("organization_id = ?", orgID).
		Preload("Violations", "status = ?", "pending").
		Find(&policies).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch policies",
		})
	}

	// Convert to API format
	var result []map[string]interface{}
	for _, p := range policies {
		var config map[string]interface{}
		json.Unmarshal([]byte(p.Config), &config)

		var violations []map[string]interface{}
		for _, v := range p.Violations {
			violations = append(violations, map[string]interface{}{
				"id":            v.ID,
				"resourceId":    v.ResourceID,
				"resourceType":  v.ResourceType,
				"cloudProvider": v.CloudProvider,
				"message":       v.Message,
				"severity":      v.Severity,
				"status":        v.Status,
				"createdAt":     v.CreatedAt,
			})
		}

		result = append(result, map[string]interface{}{
			"id":          p.ID,
			"name":        p.Name,
			"description": p.Description,
			"type":        p.Type,
			"enabled":     p.Enabled,
			"rego":        p.Rego,
			"config":      config,
			"createdAt":   p.CreatedAt,
			"updatedAt":   p.UpdatedAt,
			"violations":  violations,
		})
	}

	return c.JSON(result)
}

func (h *Handlers) GetPolicy(c *fiber.Ctx) error {
	orgID := middleware_.GetOrgID(c)
	id := c.Params("id")

	var policy models_.Policy
	if err := h.DB.Where("id = ? AND organization_id = ?", id, orgID).First(&policy).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Policy not found",
		})
	}

	var config map[string]interface{}
	json.Unmarshal([]byte(policy.Config), &config)

	return c.JSON(map[string]interface{}{
		"id":          policy.ID,
		"name":        policy.Name,
		"description": policy.Description,
		"type":        policy.Type,
		"enabled":     policy.Enabled,
		"rego":        policy.Rego,
		"config":      config,
		"createdAt":   policy.CreatedAt,
		"updatedAt":   policy.UpdatedAt,
	})
}

func (h *Handlers) CreatePolicy(c *fiber.Ctx) error {
	orgID := middleware_.GetOrgID(c)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Organization ID required",
		})
	}

	var req struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Type        string                 `json:"type"`
		Config      map[string]interface{} `json:"config"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Generate Rego policy
	rego, err := policygen_.GenerateRego(req.Type, req.Config)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to generate policy: " + err.Error(),
		})
	}

	configJSON, _ := json.Marshal(req.Config)

	policy := models_.Policy{
		OrganizationID: orgID,
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		Enabled:        true,
		Rego:           rego,
		Config:         string(configJSON),
	}

	if err := h.DB.Create(&policy).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create policy",
		})
	}

	// Reload OPA policies
	h.OPA.ReloadPolicies()

	// Create activity log
	activityLog := models_.ActivityLog{
		OrganizationID: orgID,
		Type:           "policy_created",
		Message:        "Policy '" + policy.Name + "' was created",
		Metadata:       `{"policyId":"` + policy.ID + `"}`,
	}
	h.DB.Create(&activityLog)

	return c.JSON(map[string]interface{}{
		"id":          policy.ID,
		"name":        policy.Name,
		"description": policy.Description,
		"type":        policy.Type,
		"enabled":     policy.Enabled,
		"rego":        policy.Rego,
		"config":      req.Config,
		"createdAt":   policy.CreatedAt,
		"updatedAt":   policy.UpdatedAt,
	})
}

func (h *Handlers) UpdatePolicy(c *fiber.Ctx) error {
	orgID := middleware_.GetOrgID(c)
	id := c.Params("id")

	var req struct {
		Enabled *bool `json:"enabled"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var policy models_.Policy
	if err := h.DB.Where("id = ? AND organization_id = ?", id, orgID).First(&policy).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Policy not found",
		})
	}

	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}

	if err := h.DB.Save(&policy).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update policy",
		})
	}

	// Reload OPA policies
	h.OPA.ReloadPolicies()

	return c.JSON(map[string]interface{}{
		"id":      policy.ID,
		"enabled": policy.Enabled,
	})
}

func (h *Handlers) DeletePolicy(c *fiber.Ctx) error {
	orgID := middleware_.GetOrgID(c)
	id := c.Params("id")

		if err := h.DB.Where("id = ? AND organization_id = ?", id, orgID).Delete(&models_.Policy{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete policy",
		})
	}

	// Reload OPA policies
	h.OPA.ReloadPolicies()

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handlers) ListCloudProviders(c *fiber.Ctx) error {
	orgID := middleware_.GetOrgID(c)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Organization ID required",
		})
	}

	var providers []models_.CloudProvider
	if err := h.DB.Where("organization_id = ?", orgID).Find(&providers).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch cloud providers",
		})
	}

	var result []map[string]interface{}
	for _, p := range providers {
		var credentials map[string]interface{}
		json.Unmarshal([]byte(p.Credentials), &credentials)

		result = append(result, map[string]interface{}{
			"id":             p.ID,
			"type":           p.Type,
			"name":           p.Name,
			"accountId":      p.AccountID,
			"subscriptionId": p.SubscriptionID,
			"projectId":      p.ProjectID,
			"status":         p.Status,
			"monthlySpend":   p.MonthlySpend,
			"connectedAt":    p.ConnectedAt,
			"credentials":    credentials,
		})
	}

	return c.JSON(result)
}

func (h *Handlers) GetCloudProvider(c *fiber.Ctx) error {
	orgID := middleware_.GetOrgID(c)
	id := c.Params("id")

	var provider models_.CloudProvider
	if err := h.DB.Where("id = ? AND organization_id = ?", id, orgID).First(&provider).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Cloud provider not found",
		})
	}

	var credentials map[string]interface{}
	json.Unmarshal([]byte(provider.Credentials), &credentials)

	return c.JSON(map[string]interface{}{
		"id":             provider.ID,
		"type":           provider.Type,
		"name":           provider.Name,
		"accountId":      provider.AccountID,
		"subscriptionId": provider.SubscriptionID,
		"projectId":      provider.ProjectID,
		"status":         provider.Status,
		"monthlySpend":   provider.MonthlySpend,
		"connectedAt":    provider.ConnectedAt,
		"credentials":    credentials,
	})
}

func (h *Handlers) CreateCloudProvider(c *fiber.Ctx) error {
	orgID := middleware_.GetOrgID(c)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Organization ID required",
		})
	}

	var req struct {
		Type           string                 `json:"type"`
		Name           string                 `json:"name"`
		AccountID      string                 `json:"accountId"`
		SubscriptionID string                 `json:"subscriptionId"`
		ProjectID      string                 `json:"projectId"`
		Credentials    map[string]interface{} `json:"credentials"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	credentialsJSON, _ := json.Marshal(req.Credentials)
	now := time.Now()

	provider := models_.CloudProvider{
		OrganizationID: orgID,
		Type:           req.Type,
		Name:           req.Name,
		AccountID:      req.AccountID,
		SubscriptionID: req.SubscriptionID,
		ProjectID:      req.ProjectID,
		Status:         "connected",
		Credentials:    string(credentialsJSON),
		ConnectedAt:    &now,
	}

	if err := h.DB.Create(&provider).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create cloud provider",
		})
	}

	// Create activity log
	activityLog := models_.ActivityLog{
		OrganizationID: orgID,
		Type:           "cloud_connected",
		Message:        "Cloud provider '" + provider.Name + "' (" + provider.Type + ") was connected",
		Metadata:       `{"providerId":"` + provider.ID + `"}`,
	}
	h.DB.Create(&activityLog)

	return c.JSON(map[string]interface{}{
		"id":             provider.ID,
		"type":           provider.Type,
		"name":           provider.Name,
		"accountId":      provider.AccountID,
		"subscriptionId": provider.SubscriptionID,
		"projectId":      provider.ProjectID,
		"status":         provider.Status,
		"connectedAt":   provider.ConnectedAt,
	})
}

func (h *Handlers) DeleteCloudProvider(c *fiber.Ctx) error {
	orgID := middleware_.GetOrgID(c)
	id := c.Params("id")

		if err := h.DB.Where("id = ? AND organization_id = ?", id, orgID).Delete(&models_.CloudProvider{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete cloud provider",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handlers) ListActivityLogs(c *fiber.Ctx) error {
	orgID := middleware_.GetOrgID(c)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Organization ID required",
		})
	}

	var logs []models_.ActivityLog
	if err := h.DB.Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Limit(100).
		Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch activity logs",
		})
	}

	var result []map[string]interface{}
	for _, log := range logs {
		var metadata map[string]interface{}
		json.Unmarshal([]byte(log.Metadata), &metadata)

		result = append(result, map[string]interface{}{
			"id":        log.ID,
			"type":      log.Type,
			"message":   log.Message,
			"metadata":  metadata,
			"createdAt": log.CreatedAt,
		})
	}

	return c.JSON(result)
}

func (h *Handlers) ListWebhooks(c *fiber.Ctx) error {
	orgID := middleware_.GetOrgID(c)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Organization ID required",
		})
	}

	var webhooks []models_.Webhook
	if err := h.DB.Where("organization_id = ?", orgID).Find(&webhooks).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch webhooks",
		})
	}

	var result []map[string]interface{}
	for _, w := range webhooks {
		result = append(result, map[string]interface{}{
			"id":        w.ID,
			"type":      w.Type,
			"url":       w.URL,
			"enabled":   w.Enabled,
			"createdAt": w.CreatedAt,
		})
	}

	return c.JSON(result)
}

func (h *Handlers) CreateWebhook(c *fiber.Ctx) error {
	orgID := middleware_.GetOrgID(c)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Organization ID required",
		})
	}

	var req struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	webhook := models_.Webhook{
		OrganizationID: orgID,
		Type:          req.Type,
		URL:           req.URL,
		Enabled:       true,
	}

	if err := h.DB.Create(&webhook).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create webhook",
		})
	}

	return c.JSON(webhook)
}

func (h *Handlers) DeleteWebhook(c *fiber.Ctx) error {
	orgID := middleware_.GetOrgID(c)
	id := c.Params("id")

		if err := h.DB.Where("id = ? AND organization_id = ?", id, orgID).Delete(&models_.Webhook{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete webhook",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

