"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"
import { apiRequestWithAuth } from "@/lib/api"
import { useAuth } from "@clerk/nextjs"
import { Policy } from "@/lib/types"
import { formatDate } from "@/lib/utils"
import { Plus, Shield, AlertTriangle } from "lucide-react"
import Link from "next/link"
import { useToast } from "@/hooks/use-toast"

export default function PoliciesPage() {
  const { getToken } = useAuth()
  const queryClient = useQueryClient()
  const { toast } = useToast()

  const { data: policies, isLoading } = useQuery<Policy[]>({
    queryKey: ["policies"],
    queryFn: async () => {
      const token = await getToken()
      if (!token) throw new Error("Not authenticated")
      return apiRequestWithAuth("/api/policies", token)
    },
  })

  const togglePolicy = useMutation({
    mutationFn: async ({ id, enabled }: { id: string; enabled: boolean }) => {
      const token = await getToken()
      if (!token) throw new Error("Not authenticated")
      return apiRequestWithAuth(`/api/policies/${id}`, token, {
        method: "PATCH",
        body: JSON.stringify({ enabled }),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["policies"] })
      toast({
        title: "Policy updated",
        description: "Policy status has been updated.",
      })
    },
  })

  if (isLoading) {
    return <div className="text-center py-12">Loading...</div>
  }

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Policies</h1>
          <p className="text-muted-foreground">Manage your governance policies</p>
        </div>
        <Link href="/dashboard/policies/new">
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            New Policy
          </Button>
        </Link>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {policies?.map((policy) => (
          <Card key={policy.id}>
            <CardHeader>
              <div className="flex items-center justify-between">
                <Shield className="h-5 w-5 text-primary" />
                <Switch
                  checked={policy.enabled}
                  onCheckedChange={(enabled) => togglePolicy.mutate({ id: policy.id, enabled })}
                />
              </div>
              <CardTitle>{policy.name}</CardTitle>
              <CardDescription>{policy.description}</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">Type:</span>
                  <span className="font-medium">{policy.type.replace(/_/g, " ")}</span>
                </div>
                {policy.violations && policy.violations.length > 0 && (
                  <div className="flex items-center gap-2 text-sm text-destructive">
                    <AlertTriangle className="h-4 w-4" />
                    <span>{policy.violations.length} active violations</span>
                  </div>
                )}
                <div className="text-xs text-muted-foreground pt-2">
                  Updated {formatDate(policy.updatedAt)}
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {(!policies || policies.length === 0) && (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Shield className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground mb-4">No policies yet</p>
            <Link href="/dashboard/policies/new">
              <Button>Create Your First Policy</Button>
            </Link>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

