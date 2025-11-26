export type CloudProviderType = 'aws' | 'azure' | 'gcp' | 'oci' | 'ibm'

export interface CloudProvider {
  id: string
  type: CloudProviderType
  name: string
  accountId?: string
  subscriptionId?: string
  projectId?: string
  tenancyId?: string // OCI
  compartmentId?: string // OCI
  ibmAccountId?: string // IBM Cloud
  status: 'connected' | 'disconnected' | 'error'
  connectedAt?: string
  monthlySpend?: number
  credentials?: {
    roleArn?: string
    servicePrincipalId?: string
    servicePrincipalSecret?: string
    tenantId?: string
    serviceAccountKey?: string
    ociUserId?: string // OCI
    ociFingerprint?: string // OCI
    ociPrivateKey?: string // OCI
    ibmApiKey?: string // IBM Cloud
  }
}

export const CLOUD_PROVIDER_LABELS: Record<CloudProviderType, string> = {
  aws: 'Amazon Web Services',
  azure: 'Microsoft Azure',
  gcp: 'Google Cloud Platform',
  oci: 'Oracle Cloud Infrastructure',
  ibm: 'IBM Cloud',
}

export const CLOUD_PROVIDER_SHORT_LABELS: Record<CloudProviderType, string> = {
  aws: 'AWS',
  azure: 'Azure',
  gcp: 'GCP',
  oci: 'OCI',
  ibm: 'IBM',
}

export interface Policy {
  id: string
  name: string
  description: string
  type: 'max_spend' | 'block_instance_type' | 'auto_stop_idle' | 'require_tags'
  enabled: boolean
  rego: string
  config: Record<string, any>
  createdAt: string
  updatedAt: string
  violations?: PolicyViolation[]
}

export interface PolicyViolation {
  id: string
  policyId: string
  resourceId: string
  resourceType: string
  cloudProvider: string
  message: string
  severity: 'low' | 'medium' | 'high' | 'critical'
  status: 'pending' | 'remediated' | 'ignored'
  createdAt: string
  remediatedAt?: string
}

export interface ActivityLog {
  id: string
  type: 'policy_violation' | 'remediation' | 'policy_created' | 'policy_updated' | 'cloud_connected' | 'cloud_disconnected'
  message: string
  metadata?: Record<string, any>
  createdAt: string
}

export interface DashboardStats {
  totalSpend: number
  activePolicies: number
  connectedClouds: number
  violations: number
  remediations: number
  spendByProvider: Array<{
    provider: string
    amount: number
  }>
  spendTrend: Array<{
    date: string
    amount: number
  }>
}

export interface WaitlistEntry {
  id: string
  email: string
  name?: string
  company?: string
  createdAt: string
}

export interface PolicyCategory {
  id: string
  name: string
  description: string
  icon: string
  sortOrder: number
  createdAt: string
  updatedAt: string
  templates?: PolicyTemplate[]
}

export interface PolicyTemplate {
  id: string
  categoryId: string
  name: string
  description: string
  policyType: string
  defaultConfig: string // JSON
  regoTemplate: string
  estimatedSavings: string
  difficulty: 'easy' | 'medium' | 'hard'
  requiredPermissions: string // JSON array
  tags: string // JSON array
  cloudProviders: string // JSON array
  complianceFrameworks: string // JSON array
  businessImpact: string
  usageCount: number
  createdAt: string
  updatedAt: string
}

export interface PolicyRecommendation {
  id: string
  organizationId: string
  policyTemplateId: string
  status: 'pending' | 'accepted' | 'rejected' | 'deployed'
  confidenceScore: number
  estimatedMonthlySavings: number
  recommendationReason: string
  detectedIssues: string // JSON array
  suggestedConfig: string // JSON
  priority: 'low' | 'medium' | 'high' | 'critical'
  createdAt: string
  updatedAt: string
  deployedAt?: string
  rejectedAt?: string
  rejectionReason?: string
  template?: PolicyTemplate
}

export interface RecommendationWithTemplate extends PolicyRecommendation {
  template: PolicyTemplate
}

