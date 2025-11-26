"use client"

import { useEffect, useState } from "react"
import { useAuth } from "@clerk/nextjs"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { useToast } from "@/hooks/use-toast"
import { apiRequestWithAuth } from "@/lib/utils"
import type { RecommendationWithTemplate } from "@/lib/types"
import {
  Sparkles,
  TrendingUp,
  AlertCircle,
  CheckCircle2,
  XCircle,
  Loader2,
  DollarSign,
  Gauge,
  ChevronRight,
  RefreshCw,
} from "lucide-react"
import Link from "next/link"

const PRIORITY_CONFIG = {
  critical: {
    color: "bg-red-100 text-red-800 border-red-200",
    icon: AlertCircle,
    label: "Critical",
  },
  high: {
    color: "bg-orange-100 text-orange-800 border-orange-200",
    icon: AlertCircle,
    label: "High Priority",
  },
  medium: {
    color: "bg-yellow-100 text-yellow-800 border-yellow-200",
    icon: AlertCircle,
    label: "Medium Priority",
  },
  low: {
    color: "bg-blue-100 text-blue-800 border-blue-200",
    icon: AlertCircle,
    label: "Low Priority",
  },
}

export default function RecommendationsPage() {
  const { getToken } = useAuth()
  const { toast } = useToast()

  const [recommendations, setRecommendations] = useState<RecommendationWithTemplate[]>([])
  const [loading, setLoading] = useState(true)
  const [generating, setGenerating] = useState(false)

  useEffect(() => {
    fetchRecommendations()
  }, [])

  const fetchRecommendations = async () => {
    try {
      setLoading(true)
      const token = await getToken()
      if (!token) return

      const data = await apiRequestWithAuth("/api/recommendations", token)
      setRecommendations(Array.isArray(data) ? data : [])
    } catch (error) {
      console.error("Error fetching recommendations:", error)
      toast({
        title: "Error",
        description: "Failed to load recommendations",
        variant: "destructive",
      })
    } finally {
      setLoading(false)
    }
  }

  const generateRecommendations = async () => {
    try {
      setGenerating(true)
      const token = await getToken()
      if (!token) return

      const data = await apiRequestWithAuth("/api/recommendations/generate", token, {
        method: "POST",
      })

      setRecommendations(Array.isArray(data) ? data : [])

      toast({
        title: "Success",
        description: `Generated ${data.length} policy recommendations`,
      })
    } catch (error) {
      console.error("Error generating recommendations:", error)
      toast({
        title: "Error",
        description: "Failed to generate recommendations",
        variant: "destructive",
      })
    } finally {
      setGenerating(false)
    }
  }

  const acceptRecommendation = async (id: string) => {
    try {
      const token = await getToken()
      if (!token) return

      await apiRequestWithAuth(`/api/recommendations/${id}/accept`, token, {
        method: "POST",
      })

      toast({
        title: "Success",
        description: "Recommendation accepted",
      })

      fetchRecommendations()
    } catch (error) {
      console.error("Error accepting recommendation:", error)
      toast({
        title: "Error",
        description: "Failed to accept recommendation",
        variant: "destructive",
      })
    }
  }

  const rejectRecommendation = async (id: string, reason: string) => {
    try {
      const token = await getToken()
      if (!token) return

      await apiRequestWithAuth(`/api/recommendations/${id}/reject`, token, {
        method: "POST",
        body: JSON.stringify({ reason }),
      })

      toast({
        title: "Success",
        description: "Recommendation rejected",
      })

      fetchRecommendations()
    } catch (error) {
      console.error("Error rejecting recommendation:", error)
      toast({
        title: "Error",
        description: "Failed to reject recommendation",
        variant: "destructive",
      })
    }
  }

  const parseJSON = (jsonString: string, fallback: any = []) => {
    try {
      return JSON.parse(jsonString)
    } catch {
      return fallback
    }
  }

  const pendingRecommendations = recommendations.filter((r) => r.status === "pending")
  const totalSavings = pendingRecommendations.reduce((sum, r) => sum + r.estimatedMonthlySavings, 0)

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading recommendations...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="p-8">
      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h1 className="text-3xl font-bold flex items-center gap-2">
              <Sparkles className="h-8 w-8 text-purple-600" />
              AI Policy Recommendations
            </h1>
            <p className="text-gray-600 mt-2">
              Intelligent policy suggestions based on your cloud spending patterns
            </p>
          </div>
          <Button
            onClick={generateRecommendations}
            disabled={generating}
            className="gap-2"
          >
            {generating ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Generating...
              </>
            ) : (
              <>
                <RefreshCw className="h-4 w-4" />
                Generate New Recommendations
              </>
            )}
          </Button>
        </div>

        {/* Savings Overview */}
        {pendingRecommendations.length > 0 && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-6">
            <Card className="bg-gradient-to-br from-green-50 to-emerald-50 border-green-200">
              <CardContent className="pt-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-green-700">Potential Monthly Savings</p>
                    <p className="text-3xl font-bold text-green-900">
                      ${totalSavings.toLocaleString()}
                    </p>
                  </div>
                  <DollarSign className="h-12 w-12 text-green-600" />
                </div>
              </CardContent>
            </Card>

            <Card className="bg-gradient-to-br from-blue-50 to-indigo-50 border-blue-200">
              <CardContent className="pt-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-blue-700">Pending Recommendations</p>
                    <p className="text-3xl font-bold text-blue-900">
                      {pendingRecommendations.length}
                    </p>
                  </div>
                  <Sparkles className="h-12 w-12 text-blue-600" />
                </div>
              </CardContent>
            </Card>

            <Card className="bg-gradient-to-br from-purple-50 to-pink-50 border-purple-200">
              <CardContent className="pt-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-purple-700">Avg. Confidence Score</p>
                    <p className="text-3xl font-bold text-purple-900">
                      {Math.round(
                        (pendingRecommendations.reduce((sum, r) => sum + r.confidenceScore, 0) /
                          pendingRecommendations.length) *
                          100
                      )}
                      %
                    </p>
                  </div>
                  <Gauge className="h-12 w-12 text-purple-600" />
                </div>
              </CardContent>
            </Card>
          </div>
        )}
      </div>

      {/* Recommendations List */}
      {pendingRecommendations.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <Sparkles className="h-16 w-16 text-gray-400 mx-auto mb-4" />
            <h3 className="text-xl font-semibold mb-2">No Recommendations Yet</h3>
            <p className="text-gray-600 mb-6">
              Generate AI-powered policy recommendations based on your cloud spending patterns
            </p>
            <Button onClick={generateRecommendations} disabled={generating} className="gap-2">
              {generating ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Generating...
                </>
              ) : (
                <>
                  <Sparkles className="h-4 w-4" />
                  Generate Recommendations
                </>
              )}
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {pendingRecommendations.map((rec) => {
            const PriorityIcon = PRIORITY_CONFIG[rec.priority].icon
            const detectedIssues = parseJSON(rec.detectedIssues, [])
            const cloudProviders = parseJSON(rec.template?.cloudProviders || "[]", [])

            return (
              <Card key={rec.id} className="hover:shadow-lg transition-shadow">
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-3 mb-2">
                        <CardTitle className="text-xl">{rec.template?.name}</CardTitle>
                        <Badge
                          variant="outline"
                          className={PRIORITY_CONFIG[rec.priority].color}
                        >
                          <PriorityIcon className="h-3 w-3 mr-1" />
                          {PRIORITY_CONFIG[rec.priority].label}
                        </Badge>
                        <Badge variant="outline" className="bg-purple-50 text-purple-700">
                          <Gauge className="h-3 w-3 mr-1" />
                          {Math.round(rec.confidenceScore * 100)}% Confidence
                        </Badge>
                      </div>
                      <CardDescription>{rec.template?.description}</CardDescription>
                    </div>
                  </div>
                </CardHeader>

                <CardContent>
                  <div className="space-y-4">
                    {/* Estimated Savings */}
                    <div className="bg-green-50 border border-green-200 rounded-lg p-4">
                      <div className="flex items-center gap-2 mb-1">
                        <TrendingUp className="h-5 w-5 text-green-600" />
                        <span className="font-semibold text-green-900">
                          Estimated Monthly Savings
                        </span>
                      </div>
                      <p className="text-2xl font-bold text-green-600">
                        ${rec.estimatedMonthlySavings.toLocaleString()}
                      </p>
                    </div>

                    {/* Recommendation Reason */}
                    <div>
                      <h4 className="font-semibold mb-2">Why This Policy?</h4>
                      <p className="text-sm text-gray-700">{rec.recommendationReason}</p>
                    </div>

                    {/* Detected Issues */}
                    {detectedIssues.length > 0 && (
                      <div>
                        <h4 className="font-semibold mb-2">Detected Issues</h4>
                        <ul className="space-y-1">
                          {detectedIssues.map((issue: string, idx: number) => (
                            <li key={idx} className="flex items-start gap-2 text-sm">
                              <AlertCircle className="h-4 w-4 text-orange-600 mt-0.5 flex-shrink-0" />
                              <span className="text-gray-700">{issue}</span>
                            </li>
                          ))}
                        </ul>
                      </div>
                    )}

                    {/* Cloud Providers */}
                    <div>
                      <h4 className="font-semibold mb-2 text-sm">Supported Providers</h4>
                      <div className="flex flex-wrap gap-1">
                        {cloudProviders.map((provider: string) => (
                          <Badge key={provider} variant="secondary" className="text-xs">
                            {provider.toUpperCase()}
                          </Badge>
                        ))}
                      </div>
                    </div>

                    {/* Actions */}
                    <div className="flex gap-3 pt-2">
                      <Link href={`/dashboard/policy-library/${rec.policyTemplateId}`} className="flex-1">
                        <Button className="w-full gap-2">
                          <CheckCircle2 className="h-4 w-4" />
                          View & Deploy Policy
                          <ChevronRight className="h-4 w-4" />
                        </Button>
                      </Link>
                      <Button
                        variant="outline"
                        onClick={() => acceptRecommendation(rec.id)}
                        className="gap-2"
                      >
                        <CheckCircle2 className="h-4 w-4" />
                        Accept
                      </Button>
                      <Button
                        variant="outline"
                        onClick={() =>
                          rejectRecommendation(rec.id, "Not applicable to our use case")
                        }
                        className="gap-2 text-red-600 hover:text-red-700"
                      >
                        <XCircle className="h-4 w-4" />
                        Reject
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}

      {/* Previously Accepted/Rejected */}
      {recommendations.some((r) => r.status !== "pending") && (
        <div className="mt-12">
          <h2 className="text-2xl font-bold mb-4">Previous Recommendations</h2>
          <div className="space-y-3">
            {recommendations
              .filter((r) => r.status !== "pending")
              .map((rec) => (
                <Card key={rec.id} className="opacity-60">
                  <CardContent className="py-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="font-semibold">{rec.template?.name}</p>
                        <p className="text-sm text-gray-600">
                          ${rec.estimatedMonthlySavings.toLocaleString()} potential savings
                        </p>
                      </div>
                      {rec.status === "accepted" ? (
                        <Badge className="bg-green-100 text-green-800">
                          <CheckCircle2 className="h-3 w-3 mr-1" />
                          Accepted
                        </Badge>
                      ) : (
                        <Badge className="bg-gray-100 text-gray-800">
                          <XCircle className="h-3 w-3 mr-1" />
                          Rejected
                        </Badge>
                      )}
                    </div>
                  </CardContent>
                </Card>
              ))}
          </div>
        </div>
      )}
    </div>
  )
}
