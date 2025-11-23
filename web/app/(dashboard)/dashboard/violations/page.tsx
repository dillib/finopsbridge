"use client"

import { useState, useEffect, useMemo } from "react"
import { useAuth } from "@clerk/nextjs"
import { ColumnDef } from "@tanstack/react-table"
import { DataTable, DataTableColumnHeader } from "@/components/ui/data-table"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { apiRequestWithAuth } from "@/lib/api"
import { PolicyViolation, CLOUD_PROVIDER_SHORT_LABELS, CloudProviderType } from "@/lib/types"
import { AlertTriangle, CheckCircle2, Clock, Download, Eye, MoreHorizontal, RefreshCw, XCircle } from "lucide-react"
import { useToast } from "@/hooks/use-toast"
import { formatDistanceToNow } from "date-fns"

const severityConfig = {
  low: { label: "Low", variant: "secondary" as const, icon: AlertTriangle },
  medium: { label: "Medium", variant: "default" as const, icon: AlertTriangle },
  high: { label: "High", variant: "destructive" as const, icon: AlertTriangle },
  critical: { label: "Critical", variant: "destructive" as const, icon: XCircle },
}

const statusConfig = {
  pending: { label: "Pending", variant: "secondary" as const, icon: Clock },
  remediated: { label: "Remediated", variant: "default" as const, icon: CheckCircle2 },
  ignored: { label: "Ignored", variant: "outline" as const, icon: XCircle },
}

export default function ViolationsPage() {
  const { getToken } = useAuth()
  const { toast } = useToast()
  const [violations, setViolations] = useState<PolicyViolation[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedViolation, setSelectedViolation] = useState<PolicyViolation | null>(null)
  const [detailsOpen, setDetailsOpen] = useState(false)

  const fetchViolations = async () => {
    try {
      const token = await getToken()
      if (!token) return

      const data = await apiRequestWithAuth("/api/violations", token)
      setViolations(data || [])
    } catch (error) {
      console.error("Error fetching violations:", error)
      toast({
        title: "Error",
        description: "Failed to fetch violations",
        variant: "destructive",
      })
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchViolations()
  }, [])

  const handleIgnore = async (id: string) => {
    try {
      const token = await getToken()
      if (!token) return

      await apiRequestWithAuth(`/api/violations/${id}/ignore`, token, {
        method: "POST",
      })

      setViolations((prev) =>
        prev.map((v) => (v.id === id ? { ...v, status: "ignored" } : v))
      )

      toast({
        title: "Violation ignored",
        description: "The violation has been marked as ignored",
      })
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to ignore violation",
        variant: "destructive",
      })
    }
  }

  const handleRemediate = async (id: string) => {
    try {
      const token = await getToken()
      if (!token) return

      await apiRequestWithAuth(`/api/violations/${id}/remediate`, token, {
        method: "POST",
      })

      setViolations((prev) =>
        prev.map((v) =>
          v.id === id
            ? { ...v, status: "remediated", remediatedAt: new Date().toISOString() }
            : v
        )
      )

      toast({
        title: "Remediation triggered",
        description: "The violation remediation has been initiated",
      })
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to remediate violation",
        variant: "destructive",
      })
    }
  }

  const exportToCSV = () => {
    const headers = ["ID", "Policy ID", "Resource ID", "Resource Type", "Cloud Provider", "Message", "Severity", "Status", "Created At", "Remediated At"]
    const rows = violations.map((v) => [
      v.id,
      v.policyId,
      v.resourceId,
      v.resourceType,
      v.cloudProvider,
      `"${v.message.replace(/"/g, '""')}"`,
      v.severity,
      v.status,
      v.createdAt,
      v.remediatedAt || "",
    ])

    const csv = [headers.join(","), ...rows.map((r) => r.join(","))].join("\n")
    const blob = new Blob([csv], { type: "text/csv" })
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = `violations-${new Date().toISOString().split("T")[0]}.csv`
    a.click()
    window.URL.revokeObjectURL(url)

    toast({
      title: "Export complete",
      description: "Violations have been exported to CSV",
    })
  }

  const columns: ColumnDef<PolicyViolation>[] = useMemo(
    () => [
      {
        accessorKey: "severity",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Severity" />
        ),
        cell: ({ row }) => {
          const severity = row.getValue("severity") as keyof typeof severityConfig
          const config = severityConfig[severity] || severityConfig.medium
          const Icon = config.icon
          return (
            <Badge variant={config.variant} className="gap-1">
              <Icon className="h-3 w-3" />
              {config.label}
            </Badge>
          )
        },
        filterFn: (row, id, value) => value === row.getValue(id),
      },
      {
        accessorKey: "status",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Status" />
        ),
        cell: ({ row }) => {
          const status = row.getValue("status") as keyof typeof statusConfig
          const config = statusConfig[status] || statusConfig.pending
          const Icon = config.icon
          return (
            <Badge variant={config.variant} className="gap-1">
              <Icon className="h-3 w-3" />
              {config.label}
            </Badge>
          )
        },
        filterFn: (row, id, value) => value === row.getValue(id),
      },
      {
        accessorKey: "cloudProvider",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Provider" />
        ),
        cell: ({ row }) => {
          const provider = row.getValue("cloudProvider") as CloudProviderType
          return (
            <Badge variant="outline">
              {CLOUD_PROVIDER_SHORT_LABELS[provider] || provider.toUpperCase()}
            </Badge>
          )
        },
        filterFn: (row, id, value) => value === row.getValue(id),
      },
      {
        accessorKey: "message",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Message" />
        ),
        cell: ({ row }) => (
          <div className="max-w-[300px] truncate" title={row.getValue("message")}>
            {row.getValue("message")}
          </div>
        ),
      },
      {
        accessorKey: "resourceType",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Resource Type" />
        ),
        cell: ({ row }) => (
          <span className="capitalize">{row.getValue("resourceType")}</span>
        ),
      },
      {
        accessorKey: "createdAt",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Created" />
        ),
        cell: ({ row }) => {
          const date = new Date(row.getValue("createdAt"))
          return (
            <span title={date.toLocaleString()}>
              {formatDistanceToNow(date, { addSuffix: true })}
            </span>
          )
        },
      },
      {
        id: "actions",
        cell: ({ row }) => {
          const violation = row.original
          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="h-8 w-8 p-0">
                  <span className="sr-only">Open menu</span>
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>Actions</DropdownMenuLabel>
                <DropdownMenuItem
                  onClick={() => {
                    setSelectedViolation(violation)
                    setDetailsOpen(true)
                  }}
                >
                  <Eye className="mr-2 h-4 w-4" />
                  View details
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                {violation.status === "pending" && (
                  <>
                    <DropdownMenuItem onClick={() => handleRemediate(violation.id)}>
                      <RefreshCw className="mr-2 h-4 w-4" />
                      Remediate
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => handleIgnore(violation.id)}>
                      <XCircle className="mr-2 h-4 w-4" />
                      Ignore
                    </DropdownMenuItem>
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          )
        },
      },
    ],
    []
  )

  // Stats
  const stats = useMemo(() => {
    const pending = violations.filter((v) => v.status === "pending").length
    const remediated = violations.filter((v) => v.status === "remediated").length
    const critical = violations.filter(
      (v) => v.severity === "critical" && v.status === "pending"
    ).length
    const high = violations.filter(
      (v) => v.severity === "high" && v.status === "pending"
    ).length

    return { total: violations.length, pending, remediated, critical, high }
  }, [violations])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Violations</h1>
          <p className="text-muted-foreground">
            Monitor and manage policy violations across your cloud infrastructure
          </p>
        </div>
        <Button variant="outline" onClick={fetchViolations}>
          <RefreshCw className="mr-2 h-4 w-4" />
          Refresh
        </Button>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Violations</CardTitle>
            <AlertTriangle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.total}</div>
            <p className="text-xs text-muted-foreground">All time</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Pending</CardTitle>
            <Clock className="h-4 w-4 text-yellow-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.pending}</div>
            <p className="text-xs text-muted-foreground">Requires attention</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Critical/High</CardTitle>
            <XCircle className="h-4 w-4 text-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-500">
              {stats.critical + stats.high}
            </div>
            <p className="text-xs text-muted-foreground">
              {stats.critical} critical, {stats.high} high
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Remediated</CardTitle>
            <CheckCircle2 className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-500">{stats.remediated}</div>
            <p className="text-xs text-muted-foreground">Auto or manual</p>
          </CardContent>
        </Card>
      </div>

      {/* Data Table */}
      <Card>
        <CardHeader>
          <CardTitle>All Violations</CardTitle>
          <CardDescription>
            A list of all policy violations detected in your cloud environments
          </CardDescription>
        </CardHeader>
        <CardContent>
          <DataTable
            columns={columns}
            data={violations}
            searchPlaceholder="Search violations..."
            filterableColumns={[
              {
                id: "severity",
                title: "Severity",
                options: [
                  { label: "Low", value: "low" },
                  { label: "Medium", value: "medium" },
                  { label: "High", value: "high" },
                  { label: "Critical", value: "critical" },
                ],
              },
              {
                id: "status",
                title: "Status",
                options: [
                  { label: "Pending", value: "pending" },
                  { label: "Remediated", value: "remediated" },
                  { label: "Ignored", value: "ignored" },
                ],
              },
              {
                id: "cloudProvider",
                title: "Provider",
                options: [
                  { label: "AWS", value: "aws" },
                  { label: "Azure", value: "azure" },
                  { label: "GCP", value: "gcp" },
                  { label: "OCI", value: "oci" },
                  { label: "IBM", value: "ibm" },
                ],
              },
            ]}
            onExport={exportToCSV}
            exportLabel="Export CSV"
          />
        </CardContent>
      </Card>

      {/* Details Dialog */}
      <Dialog open={detailsOpen} onOpenChange={setDetailsOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Violation Details</DialogTitle>
            <DialogDescription>
              Full details of the policy violation
            </DialogDescription>
          </DialogHeader>
          {selectedViolation && (
            <div className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <div>
                  <label className="text-sm font-medium text-muted-foreground">
                    Violation ID
                  </label>
                  <p className="font-mono text-sm">{selectedViolation.id}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-muted-foreground">
                    Policy ID
                  </label>
                  <p className="font-mono text-sm">{selectedViolation.policyId}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-muted-foreground">
                    Severity
                  </label>
                  <div className="mt-1">
                    <Badge
                      variant={
                        severityConfig[selectedViolation.severity]?.variant || "default"
                      }
                    >
                      {selectedViolation.severity}
                    </Badge>
                  </div>
                </div>
                <div>
                  <label className="text-sm font-medium text-muted-foreground">
                    Status
                  </label>
                  <div className="mt-1">
                    <Badge
                      variant={
                        statusConfig[selectedViolation.status]?.variant || "default"
                      }
                    >
                      {selectedViolation.status}
                    </Badge>
                  </div>
                </div>
                <div>
                  <label className="text-sm font-medium text-muted-foreground">
                    Cloud Provider
                  </label>
                  <p>
                    {CLOUD_PROVIDER_SHORT_LABELS[selectedViolation.cloudProvider as CloudProviderType] ||
                      selectedViolation.cloudProvider.toUpperCase()}
                  </p>
                </div>
                <div>
                  <label className="text-sm font-medium text-muted-foreground">
                    Resource Type
                  </label>
                  <p className="capitalize">{selectedViolation.resourceType}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-muted-foreground">
                    Created At
                  </label>
                  <p>{new Date(selectedViolation.createdAt).toLocaleString()}</p>
                </div>
                {selectedViolation.remediatedAt && (
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">
                      Remediated At
                    </label>
                    <p>{new Date(selectedViolation.remediatedAt).toLocaleString()}</p>
                  </div>
                )}
              </div>
              <div>
                <label className="text-sm font-medium text-muted-foreground">
                  Resource ID
                </label>
                <p className="font-mono text-sm break-all">{selectedViolation.resourceId}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-muted-foreground">
                  Message
                </label>
                <p className="text-sm mt-1 p-3 bg-muted rounded-md">
                  {selectedViolation.message}
                </p>
              </div>
            </div>
          )}
          <DialogFooter>
            {selectedViolation?.status === "pending" && (
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  onClick={() => {
                    handleIgnore(selectedViolation.id)
                    setDetailsOpen(false)
                  }}
                >
                  <XCircle className="mr-2 h-4 w-4" />
                  Ignore
                </Button>
                <Button
                  onClick={() => {
                    handleRemediate(selectedViolation.id)
                    setDetailsOpen(false)
                  }}
                >
                  <RefreshCw className="mr-2 h-4 w-4" />
                  Remediate
                </Button>
              </div>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
