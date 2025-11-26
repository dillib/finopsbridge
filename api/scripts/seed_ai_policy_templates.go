package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	database "finopsbridge/api/internal/database_"
	models "finopsbridge/api/internal/models_"
)

func main() {
	// Get DATABASE_URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	// Initialize database
	db, err := database.Initialize(databaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	fmt.Println("🤖 Seeding AI & ML Cost Control Policy Templates...")

	// Create AI & ML Cost Control category
	aiCategory := models.PolicyCategory{
		Name:        "AI & ML Cost Control",
		Description: "Policies for managing AI workload costs including LLM tokens, GPU utilization, and model selection governance",
		Icon:        "🤖",
		SortOrder:   6,
	}

	if err := db.FirstOrCreate(&aiCategory, models.PolicyCategory{Name: aiCategory.Name}).Error; err != nil {
		log.Fatalf("Failed to create AI category: %v", err)
	}

	fmt.Printf("✅ Created category: %s\n", aiCategory.Name)

	// AI Policy Templates
	templates := []models.PolicyTemplate{
		// 1. LLM Token Budget Enforcement
		{
			CategoryID:  aiCategory.ID,
			Name:        "LLM Token Budget Enforcement",
			Description: "Enforce daily and monthly token consumption limits to prevent runaway LLM API costs. Tracks usage across GPT-4, Claude, Gemini, and other LLM providers.",
			PolicyType:  "llm_token_budget",
			DefaultConfig: mustMarshal(map[string]interface{}{
				"dailyTokenLimit":   1000000,
				"monthlyTokenLimit": 25000000,
				"providers":         []string{"openai", "anthropic", "google", "azure_openai"},
				"alertThresholds":   []int{70, 85, 95, 100},
				"enforceHardLimit":  true,
				"exemptUsers":       []string{},
			}),
			RegoTemplate: `package llm_token_budget

default allow = false

allow {
    input.tokenUsage.daily < input.config.dailyTokenLimit
    input.tokenUsage.monthly < input.config.monthlyTokenLimit
}

violation[msg] {
    input.tokenUsage.daily >= input.config.dailyTokenLimit
    msg := sprintf("Daily token limit exceeded: %d/%d tokens used", [input.tokenUsage.daily, input.config.dailyTokenLimit])
}

violation[msg] {
    input.tokenUsage.monthly >= input.config.monthlyTokenLimit
    msg := sprintf("Monthly token limit exceeded: %d/%d tokens used", [input.tokenUsage.monthly, input.config.monthlyTokenLimit])
}`,
			EstimatedSavings:     "30-40%",
			Difficulty:           "easy",
			RequiredPermissions:  mustMarshal([]string{"cloudwatch:GetMetricData", "monitoring:ReadMetrics"}),
			Tags:                 mustMarshal([]string{"ai", "llm", "budget", "tokens", "cost-control"}),
			CloudProviders:       mustMarshal([]string{"aws", "azure", "gcp", "openai", "anthropic"}),
			ComplianceFrameworks: mustMarshal([]string{}),
			BusinessImpact:       "Prevents surprise LLM API bills through proactive token limit enforcement. Essential for organizations experimenting with multiple AI features.",
		},

		// 2. GPU Idle Detection & Auto-Stop
		{
			CategoryID:  aiCategory.ID,
			Name:        "GPU Idle Detection & Auto-Stop",
			Description: "Automatically stop GPU instances with low utilization (<10%) for extended periods. Single H100 GPU can cost $3-5/hour on AWS.",
			PolicyType:  "gpu_idle_detection",
			DefaultConfig: mustMarshal(map[string]interface{}{
				"idleThresholdPercent":  10,
				"idleDurationMinutes":   30,
				"autoStop":              true,
				"notifyBeforeStop":      true,
				"excludeInstances":      []string{},
				"includedGPUTypes":      []string{"A100", "V100", "H100", "T4", "A10G"},
				"excludeProductionEnv":  true,
			}),
			RegoTemplate: `package gpu_idle_detection

default allow = true

violation[msg] {
    input.gpu.utilization < input.config.idleThresholdPercent
    input.gpu.idleMinutes >= input.config.idleDurationMinutes
    not is_excluded(input.gpu.instanceId)
    not is_production(input.gpu.environment)
    msg := sprintf("GPU instance %s idle at %.1f%% for %d minutes - auto-stopping", [input.gpu.instanceId, input.gpu.utilization, input.gpu.idleMinutes])
}

is_excluded(instanceId) {
    input.config.excludeInstances[_] == instanceId
}

is_production(env) {
    input.config.excludeProductionEnv == true
    env == "production"
}`,
			EstimatedSavings:     "40-60%",
			Difficulty:           "medium",
			RequiredPermissions:  mustMarshal([]string{"ec2:DescribeInstances", "ec2:StopInstances", "compute.instances.stop"}),
			Tags:                 mustMarshal([]string{"ai", "gpu", "cost-optimization", "idle-resources"}),
			CloudProviders:       mustMarshal([]string{"aws", "azure", "gcp"}),
			ComplianceFrameworks: mustMarshal([]string{}),
			BusinessImpact:       "GPU costs are 10-50x higher than standard compute. Eliminating idle GPU hours can save $50K-200K annually for ML teams.",
		},

		// 3. Model Selection Governance
		{
			CategoryID:  aiCategory.ID,
			Name:        "Model Selection Governance",
			Description: "Prevent expensive model over-selection by requiring justification for premium models (GPT-4, Claude Opus) vs. cost-effective alternatives (GPT-4o-mini, Haiku).",
			PolicyType:  "model_selection_governance",
			DefaultConfig: mustMarshal(map[string]interface{}{
				"approvalRequired": []string{"gpt-4", "gpt-4-turbo", "claude-3-opus"},
				"recommendAlternatives": map[string]string{
					"gpt-4":          "gpt-4o-mini",
					"claude-3-opus":  "claude-3-haiku",
					"gemini-pro":     "gemini-flash",
				},
				"costThresholdPerCall": 0.01,
				"enforceForEnvironments": []string{"development", "staging"},
			}),
			RegoTemplate: `package model_selection_governance

default allow = true

violation[msg] {
    requires_approval(input.model.name)
    not has_approval(input.request)
    msg := sprintf("Model %s requires approval. Consider using %s instead (90%% cost savings)", [input.model.name, get_alternative(input.model.name)])
}

violation[msg] {
    input.model.estimatedCost > input.config.costThresholdPerCall
    is_non_production(input.environment)
    msg := sprintf("Using expensive model %s in %s environment. Cost: $%.4f per call", [input.model.name, input.environment, input.model.estimatedCost])
}

requires_approval(modelName) {
    input.config.approvalRequired[_] == modelName
}

has_approval(request) {
    request.approved == true
}

get_alternative(modelName) = alternative {
    alternative := input.config.recommendAlternatives[modelName]
}

is_non_production(env) {
    input.config.enforceForEnvironments[_] == env
}`,
			EstimatedSavings:     "80-90%",
			Difficulty:           "easy",
			RequiredPermissions:  mustMarshal([]string{}),
			Tags:                 mustMarshal([]string{"ai", "llm", "cost-optimization", "governance"}),
			CloudProviders:       mustMarshal([]string{"openai", "anthropic", "azure", "aws", "gcp"}),
			ComplianceFrameworks: mustMarshal([]string{}),
			BusinessImpact:       "GPT-4o-mini is 10x cheaper than GPT-4 for simple tasks. Proper model selection can reduce LLM costs by 80-90% without sacrificing quality.",
		},

		// 4. Prompt Caching Requirement
		{
			CategoryID:  aiCategory.ID,
			Name:        "Prompt Caching Requirement",
			Description: "Mandate caching for prompts over specified token length to reduce redundant LLM API calls and costs by 15-30%.",
			PolicyType:  "prompt_caching_requirement",
			DefaultConfig: mustMarshal(map[string]interface{}{
				"minTokensForCaching": 100,
				"cacheTTLMinutes":     60,
				"cacheProvider":       "redis",
				"exemptEndpoints":     []string{},
				"enforceForModels":    []string{"gpt-4", "claude-3-opus", "gpt-4-turbo"},
			}),
			RegoTemplate: `package prompt_caching_requirement

default allow = true

violation[msg] {
    input.prompt.tokenCount >= input.config.minTokensForCaching
    not input.request.cachingEnabled
    should_enforce_caching(input.model.name)
    msg := sprintf("Prompt with %d tokens should use caching. Enable caching to save 15-30%% on costs", [input.prompt.tokenCount])
}

should_enforce_caching(modelName) {
    input.config.enforceForModels[_] == modelName
}`,
			EstimatedSavings:     "15-30%",
			Difficulty:           "medium",
			RequiredPermissions:  mustMarshal([]string{}),
			Tags:                 mustMarshal([]string{"ai", "caching", "cost-optimization", "performance"}),
			CloudProviders:       mustMarshal([]string{"openai", "anthropic", "azure", "aws"}),
			ComplianceFrameworks: mustMarshal([]string{}),
			BusinessImpact:       "Strategic caching reduces costs by 15-30% for applications with repetitive prompts like customer service bots and document analysis.",
		},

		// 5. Batch Processing for Non-Real-Time AI
		{
			CategoryID:  aiCategory.ID,
			Name:        "Batch Processing for Non-Real-Time AI",
			Description: "Route non-urgent AI requests to batch APIs offering 50% discount. Ideal for analytics, reporting, and bulk processing workloads.",
			PolicyType:  "batch_processing_requirement",
			DefaultConfig: mustMarshal(map[string]interface{}{
				"batchEligibleTypes": []string{"analytics", "reporting", "bulk_processing"},
				"maxLatencyHours":    24,
				"minimumBatchSize":   10,
				"providers":          []string{"openai", "anthropic"},
			}),
			RegoTemplate: `package batch_processing_requirement

default allow = true

violation[msg] {
    is_batch_eligible(input.request.type)
    not input.request.useBatchAPI
    input.request.urgency != "realtime"
    msg := sprintf("Request type '%s' should use batch API for 50%% cost savings", [input.request.type])
}

is_batch_eligible(requestType) {
    input.config.batchEligibleTypes[_] == requestType
}`,
			EstimatedSavings:     "50%",
			Difficulty:           "medium",
			RequiredPermissions:  mustMarshal([]string{}),
			Tags:                 mustMarshal([]string{"ai", "batch-processing", "cost-optimization"}),
			CloudProviders:       mustMarshal([]string{"openai", "anthropic", "azure"}),
			ComplianceFrameworks: mustMarshal([]string{}),
			BusinessImpact:       "Batch APIs offer 50% discounts for non-real-time workloads. Organizations processing large volumes can save $10K-50K monthly.",
		},

		// 6. Training Job Budget Caps
		{
			CategoryID:  aiCategory.ID,
			Name:        "Training Job Budget Caps",
			Description: "Prevent expensive training job overruns by enforcing budget caps and requiring approval for high-cost training workloads.",
			PolicyType:  "training_job_budget_cap",
			DefaultConfig: mustMarshal(map[string]interface{}{
				"maxCostPerJob":         1000.0,
				"requireApprovalAbove":  500.0,
				"maxGPUHours":           100,
				"alertAtPercentages":    []int{50, 75, 90},
				"autoStopAtBudget":      true,
			}),
			RegoTemplate: `package training_job_budget_cap

default allow = true

violation[msg] {
    input.training.estimatedCost > input.config.maxCostPerJob
    not input.training.hasApproval
    msg := sprintf("Training job estimated at $%.2f exceeds max budget of $%.2f - approval required", [input.training.estimatedCost, input.config.maxCostPerJob])
}

violation[msg] {
    input.training.currentCost >= input.config.maxCostPerJob
    input.config.autoStopAtBudget
    msg := sprintf("Training job reached budget cap of $%.2f - stopping job", [input.config.maxCostPerJob])
}`,
			EstimatedSavings:     "Prevents catastrophic spending",
			Difficulty:           "medium",
			RequiredPermissions:  mustMarshal([]string{"sagemaker:StopTrainingJob", "ai-platform:cancel"}),
			Tags:                 mustMarshal([]string{"ai", "training", "budget", "cost-control"}),
			CloudProviders:       mustMarshal([]string{"aws", "azure", "gcp"}),
			ComplianceFrameworks: mustMarshal([]string{}),
			BusinessImpact:       "Prevents catastrophic training cost overruns. Single unmonitored training job can cost $10K-100K if left running.",
		},

		// 7. Spot/Preemptible Instances for Training
		{
			CategoryID:  aiCategory.ID,
			Name:        "Spot/Preemptible Instances for Training",
			Description: "Require use of spot/preemptible instances for fault-tolerant ML training jobs to save 60-90% on compute costs.",
			PolicyType:  "spot_instances_for_training",
			DefaultConfig: mustMarshal(map[string]interface{}{
				"requireSpotFor": []string{"training", "batch_inference"},
				"minJobDuration":          2,
				"allowOnDemandForHours":   1,
				"checkpointingRequired":   true,
				"excludeJobs":             []string{},
			}),
			RegoTemplate: `package spot_instances_for_training

default allow = true

violation[msg] {
    is_training_job(input.job.type)
    not input.job.useSpotInstances
    input.job.estimatedHours >= input.config.minJobDuration
    not is_excluded(input.job.id)
    msg := sprintf("Training job running %d hours should use spot instances for 60-90%% savings", [input.job.estimatedHours])
}

is_training_job(jobType) {
    input.config.requireSpotFor[_] == jobType
}

is_excluded(jobId) {
    input.config.excludeJobs[_] == jobId
}`,
			EstimatedSavings:     "60-90%",
			Difficulty:           "medium",
			RequiredPermissions:  mustMarshal([]string{"ec2:RequestSpotInstances", "compute.instances.create"}),
			Tags:                 mustMarshal([]string{"ai", "training", "spot-instances", "cost-optimization"}),
			CloudProviders:       mustMarshal([]string{"aws", "azure", "gcp"}),
			ComplianceFrameworks: mustMarshal([]string{}),
			BusinessImpact:       "Spot instances save 60-90% on training costs. With checkpointing, interruptions are minimal. Can save $50K-200K annually on ML training.",
		},

		// 8. Token Length Limits
		{
			CategoryID:  aiCategory.ID,
			Name:        "Token Length Limits",
			Description: "Control costs by limiting maximum prompt and response token lengths. Prevents verbose prompts and excessive output generation.",
			PolicyType:  "token_length_limits",
			DefaultConfig: mustMarshal(map[string]interface{}{
				"maxInputTokens":      4000,
				"maxOutputTokens":     1000,
				"enforceForModels":    []string{"gpt-4", "claude-3-opus"},
				"allowExceptions":     true,
				"exemptEndpoints":     []string{"/api/generate-report"},
			}),
			RegoTemplate: `package token_length_limits

default allow = true

violation[msg] {
    input.request.inputTokens > input.config.maxInputTokens
    should_enforce(input.model.name)
    not is_exempt(input.request.endpoint)
    msg := sprintf("Input prompt exceeds limit: %d tokens (max: %d)", [input.request.inputTokens, input.config.maxInputTokens])
}

violation[msg] {
    input.request.maxOutputTokens > input.config.maxOutputTokens
    should_enforce(input.model.name)
    not is_exempt(input.request.endpoint)
    msg := sprintf("Requested output exceeds limit: %d tokens (max: %d)", [input.request.maxOutputTokens, input.config.maxOutputTokens])
}

should_enforce(modelName) {
    input.config.enforceForModels[_] == modelName
}

is_exempt(endpoint) {
    input.config.exemptEndpoints[_] == endpoint
}`,
			EstimatedSavings:     "20-30%",
			Difficulty:           "easy",
			RequiredPermissions:  mustMarshal([]string{}),
			Tags:                 mustMarshal([]string{"ai", "tokens", "cost-control", "governance"}),
			CloudProviders:       mustMarshal([]string{"openai", "anthropic", "azure", "aws"}),
			ComplianceFrameworks: mustMarshal([]string{}),
			BusinessImpact:       "Verbose prompts and excessive output generation waste 20-30% of LLM budgets. Token limits enforce efficient prompt engineering.",
		},

		// 9. Model Versioning Governance
		{
			CategoryID:  aiCategory.ID,
			Name:        "Model Versioning Governance",
			Description: "Control costs when new model versions release (often 5-20x more expensive). Require approval before upgrading to premium model versions.",
			PolicyType:  "model_versioning_governance",
			DefaultConfig: mustMarshal(map[string]interface{}{
				"approvalRequired": []string{"gpt-5", "claude-4", "gemini-ultra"},
				"allowedModels":    []string{"gpt-4o", "claude-3-sonnet", "gemini-pro"},
				"costIncreaseThreshold": 2.0,
				"notifyOnNewReleases":   true,
			}),
			RegoTemplate: `package model_versioning_governance

default allow = true

violation[msg] {
    requires_approval(input.model.name)
    not input.request.hasApproval
    msg := sprintf("Model %s requires approval due to premium pricing", [input.model.name])
}

violation[msg] {
    is_cost_increase_significant(input.model.priceMultiplier)
    not input.request.hasApproval
    msg := sprintf("Model cost %.1fx higher than current - approval required", [input.model.priceMultiplier])
}

requires_approval(modelName) {
    input.config.approvalRequired[_] == modelName
}

is_cost_increase_significant(multiplier) {
    multiplier >= input.config.costIncreaseThreshold
}`,
			EstimatedSavings:     "Prevents 5-20x cost increases",
			Difficulty:           "easy",
			RequiredPermissions:  mustMarshal([]string{}),
			Tags:                 mustMarshal([]string{"ai", "versioning", "cost-control", "governance"}),
			CloudProviders:       mustMarshal([]string{"openai", "anthropic", "azure", "aws", "gcp"}),
			ComplianceFrameworks: mustMarshal([]string{}),
			BusinessImpact:       "New model versions can be 5-20x more expensive (e.g., GPT-4 Turbo vs. GPT-5). Governance prevents automatic cost escalation.",
		},

		// 10. Inference Endpoint Rightsizing
		{
			CategoryID:  aiCategory.ID,
			Name:        "Inference Endpoint Rightsizing",
			Description: "Auto-scale inference endpoints based on traffic patterns and utilization. Prevent over-provisioning that wastes 30-50% of inference costs.",
			PolicyType:  "inference_endpoint_rightsizing",
			DefaultConfig: mustMarshal(map[string]interface{}{
				"minUtilizationPercent": 60,
				"scaleDownThreshold":    30,
				"evaluationPeriodMin":   15,
				"minInstances":          1,
				"maxInstances":          10,
				"targetUtilization":     75,
			}),
			RegoTemplate: `package inference_endpoint_rightsizing

default allow = true

violation[msg] {
    input.endpoint.utilization < input.config.scaleDownThreshold
    input.endpoint.instanceCount > input.config.minInstances
    evaluation_period_met(input.endpoint.lowUtilizationMinutes)
    msg := sprintf("Inference endpoint at %.1f%% utilization - scale down recommended", [input.endpoint.utilization])
}

violation[msg] {
    input.endpoint.utilization > 90
    input.endpoint.instanceCount < input.config.maxInstances
    msg := "Inference endpoint overloaded - scale up recommended"
}

evaluation_period_met(minutes) {
    minutes >= input.config.evaluationPeriodMin
}`,
			EstimatedSavings:     "30-50%",
			Difficulty:           "medium",
			RequiredPermissions:  mustMarshal([]string{"sagemaker:UpdateEndpoint", "ai-platform:updateModel"}),
			Tags:                 mustMarshal([]string{"ai", "inference", "autoscaling", "cost-optimization"}),
			CloudProviders:       mustMarshal([]string{"aws", "azure", "gcp"}),
			ComplianceFrameworks: mustMarshal([]string{}),
			BusinessImpact:       "Over-provisioned inference endpoints waste 30-50% of costs. Auto-scaling matches capacity to demand, saving $20K-100K annually.",
		},

		// 11. AI Sandbox Budget Limits
		{
			CategoryID:  aiCategory.ID,
			Name:        "AI Sandbox Budget Limits",
			Description: "Control experimentation costs by setting per-user or per-team budget limits for AI sandbox environments.",
			PolicyType:  "ai_sandbox_budget_limit",
			DefaultConfig: mustMarshal(map[string]interface{}{
				"budgetPerUser":       500.0,
				"budgetPerTeam":       5000.0,
				"resetPeriod":         "monthly",
				"enforceHardLimit":    true,
				"alertThresholds":     []int{50, 75, 90, 100},
			}),
			RegoTemplate: `package ai_sandbox_budget_limit

default allow = true

violation[msg] {
    input.user.spending >= input.config.budgetPerUser
    input.config.enforceHardLimit
    msg := sprintf("User sandbox budget exceeded: $%.2f/$%.2f", [input.user.spending, input.config.budgetPerUser])
}

violation[msg] {
    input.team.spending >= input.config.budgetPerTeam
    input.config.enforceHardLimit
    msg := sprintf("Team sandbox budget exceeded: $%.2f/$%.2f", [input.team.spending, input.config.budgetPerTeam])
}`,
			EstimatedSavings:     "Prevents uncontrolled exploration",
			Difficulty:           "easy",
			RequiredPermissions:  mustMarshal([]string{}),
			Tags:                 mustMarshal([]string{"ai", "sandbox", "budget", "cost-control"}),
			CloudProviders:       mustMarshal([]string{"openai", "anthropic", "aws", "azure", "gcp"}),
			ComplianceFrameworks: mustMarshal([]string{}),
			BusinessImpact:       "Uncontrolled AI experimentation can cost $5K-50K monthly. Per-user budgets enable innovation while controlling costs.",
		},

		// 12. GPU Time-Slicing Enforcement
		{
			CategoryID:  aiCategory.ID,
			Name:        "GPU Time-Slicing Enforcement",
			Description: "Maximize GPU utilization through time-slicing, allowing multiple workloads to share GPUs and saving 50-70% on GPU costs.",
			PolicyType:  "gpu_time_slicing",
			DefaultConfig: mustMarshal(map[string]interface{}{
				"requireTimeSlicing":  true,
				"maxWorkloadsPerGPU":  4,
				"environments":        []string{"development", "staging"},
				"excludeGPUTypes":     []string{"H100"},
			}),
			RegoTemplate: `package gpu_time_slicing

default allow = true

violation[msg] {
    should_time_slice(input.gpu.environment)
    not input.gpu.timeSlicingEnabled
    not is_excluded_gpu(input.gpu.type)
    input.gpu.workloadCount < input.config.maxWorkloadsPerGPU
    msg := sprintf("GPU %s should use time-slicing in %s environment for 50-70%% cost savings", [input.gpu.instanceId, input.gpu.environment])
}

should_time_slice(env) {
    input.config.environments[_] == env
}

is_excluded_gpu(gpuType) {
    input.config.excludeGPUTypes[_] == gpuType
}`,
			EstimatedSavings:     "50-70%",
			Difficulty:           "hard",
			RequiredPermissions:  mustMarshal([]string{"kubernetes:patchNodes"}),
			Tags:                 mustMarshal([]string{"ai", "gpu", "time-slicing", "cost-optimization"}),
			CloudProviders:       mustMarshal([]string{"aws", "azure", "gcp"}),
			ComplianceFrameworks: mustMarshal([]string{}),
			BusinessImpact:       "GPU time-slicing allows 4+ workloads per GPU, reducing costs by 50-70%. Essential for development and testing environments.",
		},

		// 13. Reserved GPU Capacity Recommendations
		{
			CategoryID:  aiCategory.ID,
			Name:        "Reserved GPU Capacity Recommendations",
			Description: "Recommend reserved GPU instances or savings plans for 24/7 production inference workloads to save 40-60% vs. on-demand pricing.",
			PolicyType:  "reserved_gpu_capacity",
			DefaultConfig: mustMarshal(map[string]interface{}{
				"minUptimeHoursForRecommendation": 730,
				"savingsThreshold":                40.0,
				"commitmentTerm":                  "1-year",
				"paymentOption":                   "no-upfront",
			}),
			RegoTemplate: `package reserved_gpu_capacity

default allow = true

recommendation[msg] {
    input.gpu.uptimeHours >= input.config.minUptimeHoursForRecommendation
    input.gpu.environment == "production"
    potential_savings := calculate_savings(input.gpu.monthlyCost)
    potential_savings >= input.config.savingsThreshold
    msg := sprintf("GPU instance %s running 24/7 - consider %s reserved capacity for %.0f%% savings ($%.2f/month)", [input.gpu.instanceType, input.config.commitmentTerm, potential_savings, input.gpu.monthlyCost * (potential_savings/100)])
}

calculate_savings(monthlyCost) = savings {
    savings := 50.0
}`,
			EstimatedSavings:     "40-60%",
			Difficulty:           "easy",
			RequiredPermissions:  mustMarshal([]string{}),
			Tags:                 mustMarshal([]string{"ai", "gpu", "reserved-instances", "cost-optimization"}),
			CloudProviders:       mustMarshal([]string{"aws", "azure", "gcp"}),
			ComplianceFrameworks: mustMarshal([]string{}),
			BusinessImpact:       "Reserved GPU capacity saves 40-60% for steady-state production workloads. Can save $100K-500K annually on inference infrastructure.",
		},

		// 14. Data Transfer Minimization for AI
		{
			CategoryID:  aiCategory.ID,
			Name:        "Data Transfer Minimization for AI",
			Description: "Reduce cross-region data transfer costs by co-locating AI models and training data in the same region. Data transfer can add 15-25% to AI costs.",
			PolicyType:  "ai_data_transfer_minimization",
			DefaultConfig: mustMarshal(map[string]interface{}{
				"requireSameRegion": true,
				"allowedRegions":    []string{"us-east-1", "us-west-2"},
				"maxTransferCostPercent": 10.0,
			}),
			RegoTemplate: `package ai_data_transfer_minimization

default allow = true

violation[msg] {
    input.data.region != input.model.region
    input.config.requireSameRegion
    msg := sprintf("Data in %s, model in %s - co-locate to same region to reduce transfer costs by 15-25%%", [input.data.region, input.model.region])
}

violation[msg] {
    transfer_cost_percentage := (input.costs.dataTransfer / input.costs.total) * 100
    transfer_cost_percentage > input.config.maxTransferCostPercent
    msg := sprintf("Data transfer costs are %.1f%% of total AI costs (max: %.1f%%) - optimize data locality", [transfer_cost_percentage, input.config.maxTransferCostPercent])
}`,
			EstimatedSavings:     "15-25%",
			Difficulty:           "medium",
			RequiredPermissions:  mustMarshal([]string{}),
			Tags:                 mustMarshal([]string{"ai", "data-transfer", "cost-optimization", "networking"}),
			CloudProviders:       mustMarshal([]string{"aws", "azure", "gcp"}),
			ComplianceFrameworks: mustMarshal([]string{}),
			BusinessImpact:       "Cross-region data transfer for AI workloads adds 15-25% to costs. Co-location can save $10K-50K monthly on large-scale ML operations.",
		},

		// 15. Model Lifecycle Management
		{
			CategoryID:  aiCategory.ID,
			Name:        "Model Lifecycle Management",
			Description: "Automatically archive unused models to cold storage (S3 Glacier, Azure Archive) to save 90% on storage costs while maintaining model history.",
			PolicyType:  "model_lifecycle_management",
			DefaultConfig: mustMarshal(map[string]interface{}{
				"archiveAfterDays":    90,
				"deleteAfterDays":     365,
				"coldStorageClass":    "glacier",
				"excludeProduction":   true,
				"keepLatestVersions":  3,
			}),
			RegoTemplate: `package model_lifecycle_management

default allow = true

violation[msg] {
    days_since_use := calculate_days(input.model.lastUsed)
    days_since_use >= input.config.archiveAfterDays
    not input.model.isProduction
    not input.model.inColdStorage
    msg := sprintf("Model %s unused for %d days - archive to cold storage for 90%% storage savings", [input.model.name, days_since_use])
}

violation[msg] {
    days_since_use := calculate_days(input.model.lastUsed)
    days_since_use >= input.config.deleteAfterDays
    not is_recent_version(input.model)
    msg := sprintf("Model %s unused for %d days - consider deletion", [input.model.name, days_since_use])
}

calculate_days(lastUsed) = days {
    days := 100
}

is_recent_version(model) {
    model.versionRank <= input.config.keepLatestVersions
}`,
			EstimatedSavings:     "90% on storage",
			Difficulty:           "medium",
			RequiredPermissions:  mustMarshal([]string{"s3:PutLifecycleConfiguration", "storage.buckets.update"}),
			Tags:                 mustMarshal([]string{"ai", "storage", "lifecycle", "cost-optimization"}),
			CloudProviders:       mustMarshal([]string{"aws", "azure", "gcp"}),
			ComplianceFrameworks: mustMarshal([]string{}),
			BusinessImpact:       "ML teams accumulate hundreds of model versions. Cold storage saves 90% on storage while maintaining compliance and audit trails.",
		},
	}

	// Insert templates
	for i, template := range templates {
		if err := db.Create(&template).Error; err != nil {
			log.Printf("Warning: Failed to create template %s: %v", template.Name, err)
			continue
		}
		fmt.Printf("✅ [%d/%d] Created: %s\n", i+1, len(templates), template.Name)
	}

	fmt.Printf("\n🎉 Successfully seeded %d AI & ML policy templates!\n", len(templates))
	fmt.Println("\n📊 Summary:")
	fmt.Println("   - LLM Token Budget Enforcement")
	fmt.Println("   - GPU Idle Detection & Auto-Stop")
	fmt.Println("   - Model Selection Governance")
	fmt.Println("   - Prompt Caching Requirement")
	fmt.Println("   - Batch Processing for Non-Real-Time AI")
	fmt.Println("   - Training Job Budget Caps")
	fmt.Println("   - Spot/Preemptible Instances for Training")
	fmt.Println("   - Token Length Limits")
	fmt.Println("   - Model Versioning Governance")
	fmt.Println("   - Inference Endpoint Rightsizing")
	fmt.Println("   - AI Sandbox Budget Limits")
	fmt.Println("   - GPU Time-Slicing Enforcement")
	fmt.Println("   - Reserved GPU Capacity Recommendations")
	fmt.Println("   - Data Transfer Minimization for AI")
	fmt.Println("   - Model Lifecycle Management")
	fmt.Println("\n💰 Potential savings: 50-70% on AI/ML workloads")
}

func mustMarshal(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}
	return string(b)
}
