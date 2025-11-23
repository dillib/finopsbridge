package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	cloud "finopsbridge/api/internal/cloud_"
	config "finopsbridge/api/internal/config_"
	models "finopsbridge/api/internal/models_"
	opa "finopsbridge/api/internal/opa_"

	"gorm.io/gorm"
)

type EnforcementWorker struct {
	DB     *gorm.DB
	OPA    *opa.Engine
	Config *config.Config
}

func NewEnforcementWorker(db *gorm.DB, opaEngine *opa.Engine, cfg *config.Config) *EnforcementWorker {
	return &EnforcementWorker{
		DB:     db,
		OPA:    opaEngine,
		Config: cfg,
	}
}

func (w *EnforcementWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on start
	w.run(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.run(ctx)
		}
	}
}

func (w *EnforcementWorker) run(ctx context.Context) {
	fmt.Println("Running enforcement worker...")

	// Get all enabled policies
	var policies []models.Policy
	if err := w.DB.Where("enabled = ?", true).Find(&policies).Error; err != nil {
		fmt.Printf("Error fetching policies: %v\n", err)
		return
	}

	// Get all connected cloud providers
	var providers []models.CloudProvider
	if err := w.DB.Where("status = ?", "connected").Find(&providers).Error; err != nil {
		fmt.Printf("Error fetching cloud providers: %v\n", err)
		return
	}

	// For each provider, fetch billing data and evaluate policies
	for _, provider := range providers {
		w.processProvider(ctx, provider, policies)
	}
}

func (w *EnforcementWorker) processProvider(ctx context.Context, provider models.CloudProvider, policies []models.Policy) {
	fmt.Printf("Processing provider: %s (%s)\n", provider.Name, provider.Type)

	// Fetch billing data based on provider type
	var billingData map[string]interface{}
	var err error

	switch provider.Type {
	case "aws":
		billingData, err = cloud.FetchAWSBilling(ctx, provider, w.Config)
	case "azure":
		billingData, err = cloud.FetchAzureBilling(ctx, provider, w.Config)
	case "gcp":
		billingData, err = cloud.FetchGCPBilling(ctx, provider, w.Config)
	case "oci":
		billingData, err = cloud.FetchOCIBilling(ctx, provider, w.Config)
	case "ibm":
		billingData, err = cloud.FetchIBMBilling(ctx, provider, w.Config)
	default:
		fmt.Printf("Unknown provider type: %s\n", provider.Type)
		return
	}

	if err != nil {
		fmt.Printf("Error fetching billing data for %s: %v\n", provider.Name, err)
		return
	}

	// Update monthly spend
	if spend, ok := billingData["monthlySpend"].(float64); ok {
		provider.MonthlySpend = spend
		w.DB.Save(&provider)
	}

	// Evaluate each policy
	for _, policy := range policies {
		if policy.OrganizationID != provider.OrganizationID {
			continue
		}

		w.evaluatePolicy(ctx, policy, provider, billingData)
	}
}

func (w *EnforcementWorker) evaluatePolicy(ctx context.Context, policy models.Policy, provider models.CloudProvider, billingData map[string]interface{}) {
	// Prepare input for OPA
	input := map[string]interface{}{
		"account_id":     provider.AccountID,
		"subscription_id": provider.SubscriptionID,
		"project_id":     provider.ProjectID,
		"monthly_spend":  provider.MonthlySpend,
		"provider_type":  provider.Type,
	}

	// Merge billing data into input
	for k, v := range billingData {
		input[k] = v
	}

	// Evaluate policy with OPA
	allowed, result, err := w.OPA.EvaluatePolicy(policy.ID, input)
	if err != nil {
		fmt.Printf("Error evaluating policy %s: %v\n", policy.Name, err)
		return
	}

	if !allowed {
		// Policy violation detected
		w.handleViolation(ctx, policy, provider, result)
	}
}

func (w *EnforcementWorker) handleViolation(ctx context.Context, policy models.Policy, provider models.CloudProvider, result map[string]interface{}) {
	fmt.Printf("Policy violation detected: %s\n", policy.Name)

	// Extract violation details
	message := "Policy violation detected"
	if msg, ok := result["msg"].(string); ok {
		message = msg
	}

	// Check if violation already exists
	var existingViolation models.PolicyViolation
	err := w.DB.Where("policy_id = ? AND status = ?", policy.ID, "pending").
		First(&existingViolation).Error

	if err == gorm.ErrRecordNotFound {
		// Create new violation
		violation := models.PolicyViolation{
			PolicyID:      policy.ID,
			ResourceID:    provider.ID,
			ResourceType:  "cloud_provider",
			CloudProvider: provider.Type,
			Message:       message,
			Severity:      "high",
			Status:        "pending",
		}

		if err := w.DB.Create(&violation).Error; err != nil {
			fmt.Printf("Error creating violation: %v\n", err)
			return
		}

		// Create activity log
		activityLog := models.ActivityLog{
			OrganizationID: policy.OrganizationID,
			Type:           "policy_violation",
			Message:        fmt.Sprintf("Policy '%s' violation: %s", policy.Name, message),
			Metadata:        fmt.Sprintf(`{"policyId":"%s","violationId":"%s"}`, policy.ID, violation.ID),
		}
		w.DB.Create(&activityLog)

		// Attempt remediation based on policy type
		w.remediate(ctx, policy, provider, violation)

		// Send webhooks
		w.sendWebhooks(policy.OrganizationID, violation)
	}
}

func (w *EnforcementWorker) remediate(ctx context.Context, policy models.Policy, provider models.CloudProvider, violation models.PolicyViolation) {
	fmt.Printf("Attempting remediation for policy: %s\n", policy.Name)

	var err error
	switch policy.Type {
	case "max_spend":
		// Stop non-essential resources
		err = cloud.StopNonEssentialResources(ctx, provider, w.Config)
		case "block_instance_type":
		// Terminate oversized instances
		err = cloud.TerminateOversizedInstances(ctx, provider, w.Config)
		case "auto_stop_idle":
		// Stop idle resources
		err = cloud.StopIdleResources(ctx, provider, w.Config)
	case "require_tags":
		// Tag resources (no remediation, just notification)
		return
	}

	if err != nil {
		fmt.Printf("Remediation failed: %v\n", err)
		return
	}

	// Mark violation as remediated
	now := time.Now()
	violation.Status = "remediated"
	violation.RemediatedAt = &now
	w.DB.Save(&violation)

	// Create activity log
	activityLog := models.ActivityLog{
		OrganizationID: policy.OrganizationID,
		Type:           "remediation",
		Message:        fmt.Sprintf("Policy '%s' violation remediated", policy.Name),
		Metadata:        fmt.Sprintf(`{"policyId":"%s","violationId":"%s"}`, policy.ID, violation.ID),
	}
	w.DB.Create(&activityLog)
}

func (w *EnforcementWorker) sendWebhooks(orgID string, violation models.PolicyViolation) {
	var webhooks []models.Webhook
	if err := w.DB.Where("organization_id = ? AND enabled = ?", orgID, true).Find(&webhooks).Error; err != nil {
		fmt.Printf("Error fetching webhooks: %v\n", err)
		return
	}

	// Get policy details for webhook message
	var policy models.Policy
	if err := w.DB.Where("id = ?", violation.PolicyID).First(&policy).Error; err != nil {
		fmt.Printf("Error fetching policy for webhook: %v\n", err)
		return
	}

	for _, webhook := range webhooks {
		payload := w.formatWebhookPayload(webhook.Type, policy, violation)
		if payload == nil {
			fmt.Printf("Unknown webhook type: %s\n", webhook.Type)
			continue
		}

		if err := w.sendWebhookRequest(webhook.URL, payload); err != nil {
			fmt.Printf("Error sending webhook to %s: %v\n", webhook.URL, err)
		} else {
			fmt.Printf("Webhook sent successfully to %s\n", webhook.URL)
		}
	}
}

func (w *EnforcementWorker) formatWebhookPayload(webhookType string, policy models.Policy, violation models.PolicyViolation) []byte {
	timestamp := time.Now().Format(time.RFC3339)
	severityEmoji := map[string]string{
		"low":      "⚠️",
		"medium":   "🔶",
		"high":     "🔴",
		"critical": "🚨",
	}
	emoji := severityEmoji[violation.Severity]
	if emoji == "" {
		emoji = "⚠️"
	}

	switch webhookType {
	case "slack":
		payload := map[string]interface{}{
			"text": fmt.Sprintf("%s Policy Violation Detected", emoji),
			"blocks": []map[string]interface{}{
				{
					"type": "header",
					"text": map[string]interface{}{
						"type":  "plain_text",
						"text":  fmt.Sprintf("%s Policy Violation", emoji),
						"emoji": true,
					},
				},
				{
					"type": "section",
					"fields": []map[string]interface{}{
						{
							"type": "mrkdwn",
							"text": fmt.Sprintf("*Policy:*\n%s", policy.Name),
						},
						{
							"type": "mrkdwn",
							"text": fmt.Sprintf("*Severity:*\n%s", violation.Severity),
						},
						{
							"type": "mrkdwn",
							"text": fmt.Sprintf("*Cloud Provider:*\n%s", violation.CloudProvider),
						},
						{
							"type": "mrkdwn",
							"text": fmt.Sprintf("*Status:*\n%s", violation.Status),
						},
					},
				},
				{
					"type": "section",
					"text": map[string]interface{}{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Message:*\n%s", violation.Message),
					},
				},
				{
					"type": "context",
					"elements": []map[string]interface{}{
						{
							"type": "mrkdwn",
							"text": fmt.Sprintf("Violation ID: %s | Created: %s", violation.ID, timestamp),
						},
					},
				},
			},
		}
		jsonData, _ := json.Marshal(payload)
		return jsonData

	case "discord":
		color := map[string]int{
			"low":      0xFFFF00, // Yellow
			"medium":  0xFFA500, // Orange
			"high":    0xFF0000, // Red
			"critical": 0x8B0000, // Dark Red
		}
		colorValue := color[violation.Severity]
		if colorValue == 0 {
			colorValue = 0xFFFF00
		}

		payload := map[string]interface{}{
			"embeds": []map[string]interface{}{
				{
					"title":       fmt.Sprintf("%s Policy Violation Detected", emoji),
					"description": violation.Message,
					"color":       colorValue,
					"fields": []map[string]interface{}{
						{
							"name":   "Policy",
							"value":  policy.Name,
							"inline": true,
						},
						{
							"name":   "Severity",
							"value":  violation.Severity,
							"inline": true,
						},
						{
							"name":   "Cloud Provider",
							"value":  violation.CloudProvider,
							"inline": true,
						},
						{
							"name":   "Status",
							"value":  violation.Status,
							"inline": true,
						},
						{
							"name":   "Violation ID",
							"value":  violation.ID,
							"inline": false,
						},
					},
					"timestamp": timestamp,
				},
			},
		}
		jsonData, _ := json.Marshal(payload)
		return jsonData

	case "teams":
		payload := map[string]interface{}{
			"@type":      "MessageCard",
			"@context":   "https://schema.org/extensions",
			"summary":    fmt.Sprintf("Policy Violation: %s", policy.Name),
			"themeColor": "FF0000",
			"sections": []map[string]interface{}{
				{
					"activityTitle":    fmt.Sprintf("%s Policy Violation Detected", emoji),
					"activitySubtitle": violation.Message,
					"facts": []map[string]interface{}{
						{
							"name":  "Policy",
							"value": policy.Name,
						},
						{
							"name":  "Severity",
							"value": violation.Severity,
						},
						{
							"name":  "Cloud Provider",
							"value": violation.CloudProvider,
						},
						{
							"name":  "Status",
							"value": violation.Status,
						},
						{
							"name":  "Violation ID",
							"value": violation.ID,
						},
						{
							"name":  "Timestamp",
							"value": timestamp,
						},
					},
				},
			},
		}
		jsonData, _ := json.Marshal(payload)
		return jsonData

	default:
		// Generic JSON payload for unknown types
		payload := map[string]interface{}{
			"type":      "policy_violation",
			"policy": map[string]interface{}{
				"id":          policy.ID,
				"name":        policy.Name,
				"description": policy.Description,
			},
			"violation": map[string]interface{}{
				"id":            violation.ID,
				"resourceId":    violation.ResourceID,
				"resourceType":  violation.ResourceType,
				"cloudProvider": violation.CloudProvider,
				"message":       violation.Message,
				"severity":      violation.Severity,
				"status":        violation.Status,
				"createdAt":     violation.CreatedAt,
			},
			"timestamp": timestamp,
		}
		jsonData, _ := json.Marshal(payload)
		return jsonData
	}
}

func (w *EnforcementWorker) sendWebhookRequest(url string, payload []byte) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status code: %d", resp.StatusCode)
	}

	return nil
}

