"use client"

import { useQuery } from "@tanstack/react-query"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { apiRequestWithAuth } from "@/lib/api"
import { useAuth } from "@clerk/nextjs"
import { ActivityLog } from "@/lib/types"
import { formatDate } from "@/lib/utils"
import { Activity, Shield, Cloud, AlertTriangle, CheckCircle } from "lucide-react"

export default function ActivityPage() {
  const { getToken } = useAuth()

  const { data: activities, isLoading } = useQuery<ActivityLog[]>({
    queryKey: ["activity-log"],
    queryFn: async () => {
      const token = await getToken()
      if (!token) throw new Error("Not authenticated")
      return apiRequestWithAuth("/api/activity", token)
    },
  })

  if (isLoading) {
    return <div className="text-center py-12">Loading...</div>
  }

  const getActivityIcon = (type: string) => {
    switch (type) {
      case "policy_violation":
        return <AlertTriangle className="h-5 w-5 text-destructive" />
      case "remediation":
        return <CheckCircle className="h-5 w-5 text-green-500" />
      case "policy_created":
      case "policy_updated":
        return <Shield className="h-5 w-5 text-primary" />
      case "cloud_connected":
      case "cloud_disconnected":
        return <Cloud className="h-5 w-5 text-primary" />
      default:
        return <Activity className="h-5 w-5 text-muted-foreground" />
    }
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold">Activity Log</h1>
        <p className="text-muted-foreground">Track all policy events and remediations</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Recent Activity</CardTitle>
          <CardDescription>All events in chronological order</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {activities?.map((activity) => (
              <div
                key={activity.id}
                className="flex items-start gap-4 p-4 rounded-lg border hover:bg-accent transition-colors"
              >
                <div className="mt-1">{getActivityIcon(activity.type)}</div>
                <div className="flex-1">
                  <p className="font-medium">{activity.message}</p>
                  {activity.metadata && Object.keys(activity.metadata).length > 0 && (
                    <div className="mt-2 text-sm text-muted-foreground">
                      <pre className="text-xs bg-muted p-2 rounded">
                        {JSON.stringify(activity.metadata, null, 2)}
                      </pre>
                    </div>
                  )}
                  <p className="text-xs text-muted-foreground mt-2">
                    {formatDate(activity.createdAt)}
                  </p>
                </div>
              </div>
            ))}
          </div>

          {(!activities || activities.length === 0) && (
            <div className="text-center py-12 text-muted-foreground">
              No activity yet
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

