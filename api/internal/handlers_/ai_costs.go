package handlers

import (
	"encoding/json"
	"time"

	models "finopsbridge/api/internal/models_"

	"github.com/gofiber/fiber/v2"
)

// TrackTokenUsage records token consumption from LLM APIs
func (h *Handlers) TrackTokenUsage(c *fiber.Ctx) error {
	orgID := c.Locals("orgId").(string)

	type TokenUsageRequest struct {
		AIWorkloadID  string  `json:"aiWorkloadId"`
		Provider      string  `json:"provider"`
		ModelName     string  `json:"modelName"`
		Endpoint      string  `json:"endpoint"`
		InputTokens   int64   `json:"inputTokens"`
		OutputTokens  int64   `json:"outputTokens"`
		CachedTokens  int64   `json:"cachedTokens"`
		Cost          float64 `json:"cost"`
		RequestCount  int     `json:"requestCount"`
		Metadata      map[string]interface{} `json:"metadata"`
	}

	var req TokenUsageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	metadataJSON, _ := json.Marshal(req.Metadata)

	usage := models.TokenUsage{
		OrganizationID: orgID,
		AIWorkloadID:   req.AIWorkloadID,
		Provider:       req.Provider,
		ModelName:      req.ModelName,
		Endpoint:       req.Endpoint,
		InputTokens:    req.InputTokens,
		OutputTokens:   req.OutputTokens,
		TotalTokens:    req.InputTokens + req.OutputTokens,
		Cost:           req.Cost,
		CachedTokens:   req.CachedTokens,
		RequestCount:   req.RequestCount,
		Timestamp:      time.Now(),
		Metadata:       string(metadataJSON),
	}

	if err := h.DB.Create(&usage).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to track token usage",
		})
	}

	return c.Status(201).JSON(usage)
}

// GetTokenUsage returns token usage analytics
func (h *Handlers) GetTokenUsage(c *fiber.Ctx) error {
	orgID := c.Locals("orgId").(string)

	// Query parameters for filtering
	provider := c.Query("provider")
	modelName := c.Query("model")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	query := h.DB.Where("organization_id = ?", orgID)

	if provider != "" {
		query = query.Where("provider = ?", provider)
	}

	if modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}

	if startDate != "" {
		query = query.Where("timestamp >= ?", startDate)
	}

	if endDate != "" {
		query = query.Where("timestamp <= ?", endDate)
	}

	var usage []models.TokenUsage
	if err := query.Order("timestamp DESC").Limit(1000).Find(&usage).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch token usage",
		})
	}

	// Calculate aggregated statistics
	type TokenStats struct {
		TotalInputTokens  int64   `json:"totalInputTokens"`
		TotalOutputTokens int64   `json:"totalOutputTokens"`
		TotalTokens       int64   `json:"totalTokens"`
		TotalCost         float64 `json:"totalCost"`
		TotalRequests     int     `json:"totalRequests"`
		AvgCostPerRequest float64 `json:"avgCostPerRequest"`
		AvgTokensPerRequest float64 `json:"avgTokensPerRequest"`
		CacheSavings      float64 `json:"cacheSavings"`
	}

	stats := TokenStats{}
	for _, u := range usage {
		stats.TotalInputTokens += u.InputTokens
		stats.TotalOutputTokens += u.OutputTokens
		stats.TotalTokens += u.TotalTokens
		stats.TotalCost += u.Cost
		stats.TotalRequests += u.RequestCount
	}

	if stats.TotalRequests > 0 {
		stats.AvgCostPerRequest = stats.TotalCost / float64(stats.TotalRequests)
		stats.AvgTokensPerRequest = float64(stats.TotalTokens) / float64(stats.TotalRequests)
	}

	return c.JSON(fiber.Map{
		"usage": usage,
		"stats": stats,
	})
}

// TrackGPUMetrics records GPU utilization and costs
func (h *Handlers) TrackGPUMetrics(c *fiber.Ctx) error {
	orgID := c.Locals("orgId").(string)

	type GPUMetricsRequest struct {
		AIWorkloadID  string  `json:"aiWorkloadId"`
		CloudProvider string  `json:"cloudProvider"`
		InstanceType  string  `json:"instanceType"`
		InstanceID    string  `json:"instanceId"`
		GPUType       string  `json:"gpuType"`
		GPUCount      int     `json:"gpuCount"`
		Utilization   float64 `json:"utilization"`
		MemoryUsed    float64 `json:"memoryUsed"`
		MemoryTotal   float64 `json:"memoryTotal"`
		HourlyCost    float64 `json:"hourlyCost"`
		Status        string  `json:"status"`
		Metadata      map[string]interface{} `json:"metadata"`
	}

	var req GPUMetricsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	metadataJSON, _ := json.Marshal(req.Metadata)

	metrics := models.GPUMetrics{
		OrganizationID: orgID,
		AIWorkloadID:   req.AIWorkloadID,
		CloudProvider:  req.CloudProvider,
		InstanceType:   req.InstanceType,
		InstanceID:     req.InstanceID,
		GPUType:        req.GPUType,
		GPUCount:       req.GPUCount,
		Utilization:    req.Utilization,
		MemoryUsed:     req.MemoryUsed,
		MemoryTotal:    req.MemoryTotal,
		HourlyCost:     req.HourlyCost,
		Status:         req.Status,
		Timestamp:      time.Now(),
		Metadata:       string(metadataJSON),
	}

	if err := h.DB.Create(&metrics).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to track GPU metrics",
		})
	}

	return c.Status(201).JSON(metrics)
}

// GetGPUMetrics returns GPU utilization analytics
func (h *Handlers) GetGPUMetrics(c *fiber.Ctx) error {
	orgID := c.Locals("orgId").(string)

	cloudProvider := c.Query("provider")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	query := h.DB.Where("organization_id = ?", orgID)

	if cloudProvider != "" {
		query = query.Where("cloud_provider = ?", cloudProvider)
	}

	if startDate != "" {
		query = query.Where("timestamp >= ?", startDate)
	}

	if endDate != "" {
		query = query.Where("timestamp <= ?", endDate)
	}

	var metrics []models.GPUMetrics
	if err := query.Order("timestamp DESC").Limit(1000).Find(&metrics).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch GPU metrics",
		})
	}

	// Calculate aggregated statistics
	type GPUStats struct {
		AverageUtilization float64 `json:"averageUtilization"`
		TotalGPUHours      float64 `json:"totalGPUHours"`
		TotalCost          float64 `json:"totalCost"`
		IdleGPUHours       float64 `json:"idleGPUHours"`
		IdleCostWaste      float64 `json:"idleCostWaste"`
		UniqueInstances    int     `json:"uniqueInstances"`
	}

	stats := GPUStats{}
	instanceMap := make(map[string]bool)
	utilizationSum := 0.0
	count := 0

	for _, m := range metrics {
		instanceMap[m.InstanceID] = true
		utilizationSum += m.Utilization
		count++

		// Calculate idle waste (utilization < 10%)
		if m.Utilization < 10.0 {
			stats.IdleGPUHours += 1.0 // Assuming 1-hour intervals
			stats.IdleCostWaste += m.HourlyCost
		}

		stats.TotalCost += m.HourlyCost
	}

	stats.UniqueInstances = len(instanceMap)
	if count > 0 {
		stats.AverageUtilization = utilizationSum / float64(count)
	}
	stats.TotalGPUHours = float64(count)

	return c.JSON(fiber.Map{
		"metrics": metrics,
		"stats":   stats,
	})
}

// CreateAIWorkload creates a new AI workload for tracking
func (h *Handlers) CreateAIWorkload(c *fiber.Ctx) error {
	orgID := c.Locals("orgId").(string)

	type WorkloadRequest struct {
		CloudProvider string                 `json:"cloudProvider"`
		WorkloadType  string                 `json:"workloadType"`
		Name          string                 `json:"name"`
		ModelName     string                 `json:"modelName"`
		Environment   string                 `json:"environment"`
		Metadata      map[string]interface{} `json:"metadata"`
	}

	var req WorkloadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	metadataJSON, _ := json.Marshal(req.Metadata)

	workload := models.AIWorkload{
		OrganizationID: orgID,
		CloudProvider:  req.CloudProvider,
		WorkloadType:   req.WorkloadType,
		Name:           req.Name,
		ModelName:      req.ModelName,
		Environment:    req.Environment,
		Status:         "active",
		StartedAt:      time.Now(),
		Metadata:       string(metadataJSON),
	}

	if err := h.DB.Create(&workload).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create AI workload",
		})
	}

	h.logActivity(orgID, "ai_workload_created", "Created AI workload: "+workload.Name, nil)

	return c.Status(201).JSON(workload)
}

// ListAIWorkloads returns all AI workloads for an organization
func (h *Handlers) ListAIWorkloads(c *fiber.Ctx) error {
	orgID := c.Locals("orgId").(string)

	var workloads []models.AIWorkload
	if err := h.DB.Where("organization_id = ?", orgID).Order("created_at DESC").Find(&workloads).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch AI workloads",
		})
	}

	return c.JSON(workloads)
}

// CreateAIBudget creates a new AI budget control
func (h *Handlers) CreateAIBudget(c *fiber.Ctx) error {
	orgID := c.Locals("orgId").(string)

	type BudgetRequest struct {
		Name             string                 `json:"name"`
		BudgetType       string                 `json:"budgetType"`
		Period           string                 `json:"period"`
		LimitValue       float64                `json:"limitValue"`
		AlertThresholds  []int                  `json:"alertThresholds"`
		Scope            map[string]interface{} `json:"scope"`
	}

	var req BudgetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	thresholdsJSON, _ := json.Marshal(req.AlertThresholds)
	scopeJSON, _ := json.Marshal(req.Scope)

	budget := models.AIBudget{
		OrganizationID:  orgID,
		Name:            req.Name,
		BudgetType:      req.BudgetType,
		Period:          req.Period,
		LimitValue:      req.LimitValue,
		CurrentUsage:    0,
		AlertThresholds: string(thresholdsJSON),
		Scope:           string(scopeJSON),
		Enabled:         true,
		LastResetAt:     time.Now(),
	}

	if err := h.DB.Create(&budget).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create AI budget",
		})
	}

	h.logActivity(orgID, "ai_budget_created", "Created AI budget: "+budget.Name, nil)

	return c.Status(201).JSON(budget)
}

// ListAIBudgets returns all AI budgets for an organization
func (h *Handlers) ListAIBudgets(c *fiber.Ctx) error {
	orgID := c.Locals("orgId").(string)

	var budgets []models.AIBudget
	if err := h.DB.Where("organization_id = ?", orgID).Order("created_at DESC").Find(&budgets).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch AI budgets",
		})
	}

	// Calculate budget status
	type BudgetResponse struct {
		models.AIBudget
		PercentUsed      float64 `json:"percentUsed"`
		RemainingBudget  float64 `json:"remainingBudget"`
		IsOverBudget     bool    `json:"isOverBudget"`
	}

	var responses []BudgetResponse
	for _, budget := range budgets {
		percentUsed := 0.0
		if budget.LimitValue > 0 {
			percentUsed = (budget.CurrentUsage / budget.LimitValue) * 100
		}

		responses = append(responses, BudgetResponse{
			AIBudget:        budget,
			PercentUsed:     percentUsed,
			RemainingBudget: budget.LimitValue - budget.CurrentUsage,
			IsOverBudget:    budget.CurrentUsage >= budget.LimitValue,
		})
	}

	return c.JSON(responses)
}

// GetAIDashboard returns comprehensive AI cost dashboard data
func (h *Handlers) GetAIDashboard(c *fiber.Ctx) error {
	orgID := c.Locals("orgId").(string)

	// Get date range (default: last 30 days)
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	// Token usage summary
	var tokenUsage []models.TokenUsage
	h.DB.Where("organization_id = ? AND timestamp >= ?", orgID, startDate).Find(&tokenUsage)

	tokenStats := map[string]interface{}{
		"totalTokens": int64(0),
		"totalCost":   0.0,
		"totalRequests": 0,
	}

	for _, u := range tokenUsage {
		tokenStats["totalTokens"] = tokenStats["totalTokens"].(int64) + u.TotalTokens
		tokenStats["totalCost"] = tokenStats["totalCost"].(float64) + u.Cost
		tokenStats["totalRequests"] = tokenStats["totalRequests"].(int) + u.RequestCount
	}

	// GPU metrics summary
	var gpuMetrics []models.GPUMetrics
	h.DB.Where("organization_id = ? AND timestamp >= ?", orgID, startDate).Find(&gpuMetrics)

	gpuStats := map[string]interface{}{
		"averageUtilization": 0.0,
		"totalGPUHours":      float64(len(gpuMetrics)),
		"totalCost":          0.0,
		"idleWaste":          0.0,
	}

	utilizationSum := 0.0
	for _, m := range gpuMetrics {
		utilizationSum += m.Utilization
		gpuStats["totalCost"] = gpuStats["totalCost"].(float64) + m.HourlyCost
		if m.Utilization < 10.0 {
			gpuStats["idleWaste"] = gpuStats["idleWaste"].(float64) + m.HourlyCost
		}
	}

	if len(gpuMetrics) > 0 {
		gpuStats["averageUtilization"] = utilizationSum / float64(len(gpuMetrics))
	}

	// Active workloads
	var workloads []models.AIWorkload
	h.DB.Where("organization_id = ? AND status = ?", orgID, "active").Find(&workloads)

	// Budget summary
	var budgets []models.AIBudget
	h.DB.Where("organization_id = ? AND enabled = ?", orgID, true).Find(&budgets)

	budgetAlerts := 0
	for _, b := range budgets {
		if b.CurrentUsage >= b.LimitValue*0.9 {
			budgetAlerts++
		}
	}

	return c.JSON(fiber.Map{
		"tokenUsage": tokenStats,
		"gpuMetrics": gpuStats,
		"workloads": map[string]interface{}{
			"active": len(workloads),
			"total":  len(workloads),
		},
		"budgets": map[string]interface{}{
			"total":  len(budgets),
			"alerts": budgetAlerts,
		},
		"totalAICost": tokenStats["totalCost"].(float64) + gpuStats["totalCost"].(float64),
	})
}
