export interface CloudProvider {
  id: string
  type: 'aws' | 'azure' | 'gcp'
  name: string
  accountId?: string
  subscriptionId?: string
  projectId?: string
  status: 'connected' | 'disconnected' | 'error'
  connectedAt?: string
  monthlySpend?: number
  credentials?: {
    roleArn?: string
    servicePrincipalId?: string
    serviceAccountKey?: string
  }
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

