"use client"

import { useQuery } from "@tanstack/react-query"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { apiRequestWithAuth } from "@/lib/api"
import { useAuth } from "@clerk/nextjs"
import { DashboardStats, PolicyViolation, CLOUD_PROVIDER_SHORT_LABELS, CloudProviderType } from "@/lib/types"
import { formatCurrency } from "@/lib/utils"
import {
  Cloud,
  Shield,
  AlertTriangle,
  TrendingUp,
  TrendingDown,
  CheckCircle2,
  Clock,
  ArrowRight,
  RefreshCw,
  DollarSign,
  Zap,
  Target,
} from "lucide-react"
import { SpendChart } from "@/components/dashboard/spend-chart"
import { ProviderSpend } from "@/components/dashboard/provider-spend"
import Link from "next/link"
import { formatDistanceToNow } from "date-fns"

export default function DashboardPage() {
  const { getToken } = useAuth()

  const { data: stats, isLoading, refetch } = useQuery<DashboardStats>({
    queryKey: ["dashboard-stats"],
    queryFn: async () => {
      const token = await getToken()
      if (!token) throw new Error("Not authenticated")
      return apiRequestWithAuth("/api/dashboard/stats", token)
    },
  })

  const { data: recentViolations } = useQuery<PolicyViolation[]>({
    queryKey: ["recent-violations"],
    queryFn: async () => {
      const token = await getToken()
      if (!token) throw new Error("Not authenticated")
      return apiRequestWithAuth("/api/violations?limit=5", token)
    },
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-96">
        <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  // Calculate savings and efficiency metrics (mock for now - would come from API)
  const estimatedMonthlySavings = (stats?.totalSpend || 0) * 0.15
  const complianceScore = stats?.activePolicies ? Math.min(100, 70 + (stats.activePolicies * 5)) : 70
  const pendingViolations = recentViolations?.filter(v => v.status === "pending").length || 0

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Dashboard</h1>
          <p className="text-muted-foreground">Overview of your cloud spend and governance</p>
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          <RefreshCw className="mr-2 h-4 w-4" />
          Refresh
        </Button>
      </div>

      {/* Primary Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Spend</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {stats ? formatCurrency(stats.totalSpend) : "$0.00"}
            </div>
            <div className="flex items-center gap-1 text-xs text-muted-foreground">
              <TrendingUp className="h-3 w-3 text-green-500" />
              <span>This month across all providers</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Estimated Savings</CardTitle>
            <Zap className="h-4 w-4 text-yellow-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">
              {formatCurrency(estimatedMonthlySavings)}
            </div>
            <div className="flex items-center gap-1 text-xs text-muted-foreground">
              <TrendingDown className="h-3 w-3 text-green-500" />
              <span>Potential monthly savings</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Policies</CardTitle>
            <Shield className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.activePolicies || 0}</div>
            <div className="flex items-center gap-1 text-xs text-muted-foreground">
              <CheckCircle2 className="h-3 w-3 text-green-500" />
              <span>Enforcing governance rules</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Connected Clouds</CardTitle>
            <Cloud className="h-4 w-4 text-purple-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.connectedClouds || 0}</div>
            <div className="flex items-center gap-1 text-xs text-muted-foreground">
              <span>AWS, Azure, GCP, OCI, IBM</span>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Compliance & Violations Row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Compliance Score</CardTitle>
            <Target className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center justify-between">
              <div className="text-3xl font-bold">{complianceScore}%</div>
              <Badge variant={complianceScore >= 80 ? "default" : complianceScore >= 60 ? "secondary" : "destructive"}>
                {complianceScore >= 80 ? "Good" : complianceScore >= 60 ? "Fair" : "Needs Attention"}
              </Badge>
            </div>
            <Progress value={complianceScore} className="h-2" />
            <p className="text-xs text-muted-foreground">
              Based on active policies and violation resolution
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Pending Violations</CardTitle>
            <AlertTriangle className="h-4 w-4 text-red-500" />
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between">
              <div className="text-3xl font-bold text-red-500">{pendingViolations}</div>
              <Link href="/dashboard/violations">
                <Button variant="outline" size="sm">
                  View All
                  <ArrowRight className="ml-2 h-4 w-4" />
                </Button>
              </Link>
            </div>
            <p className="text-xs text-muted-foreground mt-2">
              {pendingViolations > 0
                ? "Requires immediate attention"
                : "All violations addressed"}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Auto-Remediations</CardTitle>
            <CheckCircle2 className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between">
              <div className="text-3xl font-bold text-green-600">{stats?.remediations || 0}</div>
              <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                This Month
              </Badge>
            </div>
            <p className="text-xs text-muted-foreground mt-2">
              Automatic policy enforcements
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <CardTitle>Spend Trend</CardTitle>
            <CardDescription>Monthly spending over time across all providers</CardDescription>
          </CardHeader>
          <CardContent>
            <SpendChart data={stats?.spendTrend || []} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Spend by Provider</CardTitle>
            <CardDescription>Breakdown of costs by cloud provider</CardDescription>
          </CardHeader>
          <CardContent>
            <ProviderSpend data={stats?.spendByProvider || []} />
          </CardContent>
        </Card>
      </div>

      {/* Recent Violations */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Recent Violations</CardTitle>
              <CardDescription>Latest policy violations detected</CardDescription>
            </div>
            <Link href="/dashboard/violations">
              <Button variant="outline" size="sm">
                View All
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            </Link>
          </div>
        </CardHeader>
        <CardContent>
          {recentViolations && recentViolations.length > 0 ? (
            <div className="space-y-4">
              {recentViolations.slice(0, 5).map((violation) => (
                <div
                  key={violation.id}
                  className="flex items-center justify-between p-4 border rounded-lg hover:bg-muted/50 transition-colors"
                >
                  <div className="flex items-center gap-4">
                    <div className={`p-2 rounded-full ${
                      violation.severity === "critical" || violation.severity === "high"
                        ? "bg-red-100 text-red-600"
                        : violation.severity === "medium"
                        ? "bg-yellow-100 text-yellow-600"
                        : "bg-gray-100 text-gray-600"
                    }`}>
                      <AlertTriangle className="h-4 w-4" />
                    </div>
                    <div>
                      <p className="font-medium text-sm">{violation.message}</p>
                      <div className="flex items-center gap-2 mt-1">
                        <Badge variant="outline" className="text-xs">
                          {CLOUD_PROVIDER_SHORT_LABELS[violation.cloudProvider as CloudProviderType] ||
                            violation.cloudProvider.toUpperCase()}
                        </Badge>
                        <span className="text-xs text-muted-foreground">
                          {formatDistanceToNow(new Date(violation.createdAt), { addSuffix: true })}
                        </span>
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge
                      variant={
                        violation.status === "pending"
                          ? "secondary"
                          : violation.status === "remediated"
                          ? "default"
                          : "outline"
                      }
                    >
                      {violation.status === "pending" && <Clock className="h-3 w-3 mr-1" />}
                      {violation.status === "remediated" && <CheckCircle2 className="h-3 w-3 mr-1" />}
                      {violation.status}
                    </Badge>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <CheckCircle2 className="h-12 w-12 text-green-500 mb-4" />
              <p className="text-lg font-medium">No violations detected</p>
              <p className="text-sm text-muted-foreground">
                Your cloud infrastructure is compliant with all active policies
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Link href="/dashboard/policies/new">
          <Card className="cursor-pointer hover:bg-muted/50 transition-colors">
            <CardContent className="flex items-center gap-4 p-6">
              <div className="p-3 rounded-full bg-primary/10">
                <Shield className="h-6 w-6 text-primary" />
              </div>
              <div>
                <p className="font-medium">Create Policy</p>
                <p className="text-sm text-muted-foreground">Add new governance rules</p>
              </div>
              <ArrowRight className="h-5 w-5 ml-auto text-muted-foreground" />
            </CardContent>
          </Card>
        </Link>

        <Link href="/dashboard/settings">
          <Card className="cursor-pointer hover:bg-muted/50 transition-colors">
            <CardContent className="flex items-center gap-4 p-6">
              <div className="p-3 rounded-full bg-purple-500/10">
                <Cloud className="h-6 w-6 text-purple-500" />
              </div>
              <div>
                <p className="font-medium">Connect Cloud</p>
                <p className="text-sm text-muted-foreground">Add cloud provider</p>
              </div>
              <ArrowRight className="h-5 w-5 ml-auto text-muted-foreground" />
            </CardContent>
          </Card>
        </Link>

        <Link href="/dashboard/activity">
          <Card className="cursor-pointer hover:bg-muted/50 transition-colors">
            <CardContent className="flex items-center gap-4 p-6">
              <div className="p-3 rounded-full bg-green-500/10">
                <TrendingUp className="h-6 w-6 text-green-500" />
              </div>
              <div>
                <p className="font-medium">View Activity</p>
                <p className="text-sm text-muted-foreground">See recent actions</p>
              </div>
              <ArrowRight className="h-5 w-5 ml-auto text-muted-foreground" />
            </CardContent>
          </Card>
        </Link>
      </div>
    </div>
  )
}
