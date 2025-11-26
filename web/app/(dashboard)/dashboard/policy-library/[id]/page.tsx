"use client"

import { useEffect, useState } from "react"
import { useAuth } from "@clerk/nextjs"
import { useParams, useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { useToast } from "@/hooks/use-toast"
import { apiRequestWithAuth } from "@/lib/api"
import type { PolicyTemplate } from "@/lib/types"
import {
  ArrowLeft,
  Rocket,
  Shield,
  TrendingUp,
  Award,
  Code,
  Settings,
  CheckCircle2,
  AlertCircle,
} from "lucide-react"
import Link from "next/link"

export default function TemplateDetailsPage() {
  const params = useParams()
  const router = useRouter()
  const { getToken } = useAuth()
  const { toast } = useToast()

  const [template, setTemplate] = useState<PolicyTemplate | null>(null)
  const [loading, setLoading] = useState(true)
  const [deploying, setDeploying] = useState(false)

  // Form state
  const [policyName, setPolicyName] = useState("")
  const [policyDescription, setPolicyDescription] = useState("")
  const [configJSON, setConfigJSON] = useState("")

  useEffect(() => {
    fetchTemplate()
  }, [params.id])

  const fetchTemplate = async () => {
    try {
      setLoading(true)
      const token = await getToken()
      if (!token) return

      const data = await apiRequestWithAuth(`/api/policy-templates/${params.id}`, token)
      setTemplate(data)

      // Pre-fill form
      setPolicyName(data.name)
      setPolicyDescription(data.description)
      setConfigJSON(JSON.stringify(JSON.parse(data.defaultConfig || "{}"), null, 2))
    } catch (error) {
      console.error("Error fetching template:", error)
      toast({
        title: "Error",
        description: "Failed to load policy template",
        variant: "destructive",
      })
    } finally {
      setLoading(false)
    }
  }

  const deployPolicy = async () => {
    try {
      setDeploying(true)

      // Validate JSON
      let config
      try {
        config = JSON.parse(configJSON)
      } catch {
        toast({
          title: "Invalid Configuration",
          description: "Please enter valid JSON configuration",
          variant: "destructive",
        })
        return
      }

      const token = await getToken()
      if (!token) return

      await apiRequestWithAuth(`/api/policy-templates/${params.id}/deploy`, token, {
        method: "POST",
        body: JSON.stringify({
          name: policyName,
          description: policyDescription,
          config,
        }),
      })

      toast({
        title: "Success!",
        description: `Policy "${policyName}" has been deployed`,
      })

      router.push("/dashboard/policies")
    } catch (error) {
      console.error("Error deploying policy:", error)
      toast({
        title: "Deployment Failed",
        description: "Failed to deploy policy. Please try again.",
        variant: "destructive",
      })
    } finally {
      setDeploying(false)
    }
  }

  const parseJSON = (jsonString: string, fallback: any = []) => {
    try {
      return JSON.parse(jsonString)
    } catch {
      return fallback
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading template...</p>
        </div>
      </div>
    )
  }

  if (!template) {
    return (
      <div className="p-8">
        <Card>
          <CardContent className="py-12 text-center">
            <AlertCircle className="h-16 w-16 text-red-500 mx-auto mb-4" />
            <h3 className="text-xl font-semibold mb-2">Template Not Found</h3>
            <Link href="/dashboard/policy-library">
              <Button variant="outline" className="gap-2 mt-4">
                <ArrowLeft className="h-4 w-4" />
                Back to Library
              </Button>
            </Link>
          </CardContent>
        </Card>
      </div>
    )
  }

  const cloudProviders = parseJSON(template.cloudProviders, [])
  const tags = parseJSON(template.tags, [])
  const complianceFrameworks = parseJSON(template.complianceFrameworks, [])
  const requiredPermissions = parseJSON(template.requiredPermissions, [])

  return (
    <div className="p-8">
      {/* Header */}
      <div className="mb-6">
        <Link href="/dashboard/policy-library">
          <Button variant="ghost" className="gap-2 mb-4">
            <ArrowLeft className="h-4 w-4" />
            Back to Library
          </Button>
        </Link>

        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-3xl font-bold">{template.name}</h1>
            <p className="text-gray-600 mt-2">{template.description}</p>
          </div>
          <Badge
            variant="outline"
            className={
              template.difficulty === "easy"
                ? "bg-green-100 text-green-800 border-green-200"
                : template.difficulty === "medium"
                ? "bg-yellow-100 text-yellow-800 border-yellow-200"
                : "bg-red-100 text-red-800 border-red-200"
            }
          >
            {template.difficulty.charAt(0).toUpperCase() + template.difficulty.slice(1)}
          </Badge>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left Column - Template Details */}
        <div className="lg:col-span-2 space-y-6">
          {/* Overview */}
          <Card>
            <CardHeader>
              <CardTitle>Overview</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Estimated Savings */}
              <div className="bg-green-50 border border-green-200 rounded-lg p-4">
                <div className="flex items-center gap-2 mb-1">
                  <TrendingUp className="h-5 w-5 text-green-600" />
                  <span className="font-semibold text-green-900">Estimated Savings</span>
                </div>
                <p className="text-xl font-bold text-green-600">{template.estimatedSavings}</p>
              </div>

              {/* Business Impact */}
              <div>
                <h4 className="font-semibold mb-2 flex items-center gap-2">
                  <Award className="h-4 w-4" />
                  Business Impact
                </h4>
                <p className="text-gray-700">{template.businessImpact}</p>
              </div>

              {/* Cloud Providers */}
              <div>
                <h4 className="font-semibold mb-2">Supported Cloud Providers</h4>
                <div className="flex flex-wrap gap-2">
                  {cloudProviders.map((provider: string) => (
                    <Badge key={provider} variant="secondary">
                      {provider.toUpperCase()}
                    </Badge>
                  ))}
                </div>
              </div>

              {/* Tags */}
              <div>
                <h4 className="font-semibold mb-2">Tags</h4>
                <div className="flex flex-wrap gap-2">
                  {tags.map((tag: string) => (
                    <Badge
                      key={tag}
                      variant="outline"
                      className="bg-blue-50 text-blue-700 border-blue-200"
                    >
                      {tag}
                    </Badge>
                  ))}
                </div>
              </div>

              {/* Compliance */}
              {complianceFrameworks.length > 0 && (
                <div>
                  <h4 className="font-semibold mb-2 flex items-center gap-2">
                    <Shield className="h-4 w-4" />
                    Compliance Frameworks
                  </h4>
                  <div className="flex flex-wrap gap-2">
                    {complianceFrameworks.map((framework: string) => (
                      <Badge
                        key={framework}
                        variant="outline"
                        className="bg-purple-50 text-purple-700 border-purple-200"
                      >
                        {framework.toUpperCase()}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}

              {/* Required Permissions */}
              {requiredPermissions.length > 0 && (
                <div>
                  <h4 className="font-semibold mb-2">Required Permissions</h4>
                  <ul className="space-y-1">
                    {requiredPermissions.map((permission: string, idx: number) => (
                      <li key={idx} className="flex items-center gap-2 text-sm text-gray-700">
                        <CheckCircle2 className="h-4 w-4 text-green-600" />
                        <code className="bg-gray-100 px-2 py-1 rounded">{permission}</code>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </CardContent>
          </Card>

          {/* OPA Rego Policy */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Code className="h-5 w-5" />
                OPA Rego Policy
              </CardTitle>
              <CardDescription>
                Open Policy Agent policy that will be evaluated
              </CardDescription>
            </CardHeader>
            <CardContent>
              <pre className="bg-gray-900 text-green-400 p-4 rounded-lg overflow-x-auto text-sm">
                <code>{template.regoTemplate}</code>
              </pre>
            </CardContent>
          </Card>
        </div>

        {/* Right Column - Deployment Wizard */}
        <div className="space-y-6">
          <Card className="sticky top-4">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Rocket className="h-5 w-5" />
                Deploy Policy
              </CardTitle>
              <CardDescription>
                Customize and deploy this policy to your organization
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Policy Name */}
              <div>
                <Label htmlFor="policyName">Policy Name</Label>
                <Input
                  id="policyName"
                  value={policyName}
                  onChange={(e) => setPolicyName(e.target.value)}
                  placeholder="Enter policy name"
                />
              </div>

              {/* Policy Description */}
              <div>
                <Label htmlFor="policyDescription">Description</Label>
                <Textarea
                  id="policyDescription"
                  value={policyDescription}
                  onChange={(e) => setPolicyDescription(e.target.value)}
                  placeholder="Enter policy description"
                  rows={3}
                />
              </div>

              {/* Configuration */}
              <div>
                <Label htmlFor="configJSON">Configuration (JSON)</Label>
                <Textarea
                  id="configJSON"
                  value={configJSON}
                  onChange={(e) => setConfigJSON(e.target.value)}
                  placeholder='{"threshold": 1000}'
                  rows={8}
                  className="font-mono text-sm"
                />
                <p className="text-xs text-gray-500 mt-1">
                  Edit the default configuration to match your requirements
                </p>
              </div>

              {/* Deploy Button */}
              <Button
                onClick={deployPolicy}
                disabled={deploying || !policyName}
                className="w-full gap-2"
              >
                {deploying ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                    Deploying...
                  </>
                ) : (
                  <>
                    <Rocket className="h-4 w-4" />
                    Deploy Policy
                  </>
                )}
              </Button>

              <p className="text-xs text-center text-gray-500">
                Policy will be activated immediately after deployment
              </p>
            </CardContent>
          </Card>

          {/* Usage Stats */}
          <Card>
            <CardContent className="pt-6">
              <div className="text-center">
                <p className="text-sm text-gray-600">Times Deployed</p>
                <p className="text-3xl font-bold text-purple-600">{template.usageCount}</p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
