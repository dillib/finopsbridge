"use client"

import { useQuery } from "@tanstack/react-query"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { apiRequestWithAuth } from "@/lib/api"
import { useAuth } from "@clerk/nextjs"
import { CloudProvider } from "@/lib/types"
import { formatCurrency, formatDate } from "@/lib/utils"
import { Plus, Cloud, CheckCircle, XCircle, AlertCircle } from "lucide-react"
import Link from "next/link"

export default function CloudsPage() {
  const { getToken } = useAuth()

  const { data: clouds, isLoading } = useQuery<CloudProvider[]>({
    queryKey: ["cloud-providers"],
    queryFn: async () => {
      const token = await getToken()
      if (!token) throw new Error("Not authenticated")
      return apiRequestWithAuth("/api/cloud-providers", token)
    },
  })

  if (isLoading) {
    return <div className="text-center py-12">Loading...</div>
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "connected":
        return <CheckCircle className="h-5 w-5 text-green-500" />
      case "error":
        return <AlertCircle className="h-5 w-5 text-destructive" />
      default:
        return <XCircle className="h-5 w-5 text-muted-foreground" />
    }
  }

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Cloud Providers</h1>
          <p className="text-muted-foreground">Manage your cloud provider connections</p>
        </div>
        <Link href="/dashboard/settings">
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            Connect Provider
          </Button>
        </Link>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {clouds?.map((cloud) => (
          <Card key={cloud.id}>
            <CardHeader>
              <div className="flex items-center justify-between">
                <Cloud className="h-6 w-6 text-primary" />
                {getStatusIcon(cloud.status)}
              </div>
              <CardTitle>{cloud.name}</CardTitle>
              <CardDescription className="uppercase">{cloud.type}</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                {cloud.accountId && (
                  <div className="text-sm">
                    <span className="text-muted-foreground">Account ID: </span>
                    <span className="font-mono">{cloud.accountId}</span>
                  </div>
                )}
                {cloud.subscriptionId && (
                  <div className="text-sm">
                    <span className="text-muted-foreground">Subscription ID: </span>
                    <span className="font-mono">{cloud.subscriptionId}</span>
                  </div>
                )}
                {cloud.projectId && (
                  <div className="text-sm">
                    <span className="text-muted-foreground">Project ID: </span>
                    <span className="font-mono">{cloud.projectId}</span>
                  </div>
                )}
                {cloud.monthlySpend !== undefined && (
                  <div className="text-sm pt-2">
                    <span className="text-muted-foreground">Monthly Spend: </span>
                    <span className="font-semibold">{formatCurrency(cloud.monthlySpend)}</span>
                  </div>
                )}
                {cloud.connectedAt && (
                  <div className="text-xs text-muted-foreground pt-2">
                    Connected {formatDate(cloud.connectedAt)}
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {(!clouds || clouds.length === 0) && (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Cloud className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground mb-4">No cloud providers connected</p>
            <Link href="/dashboard/settings">
              <Button>Connect Your First Provider</Button>
            </Link>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

