"use client"

import { useState, useMemo } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { ColumnDef } from "@tanstack/react-table"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"
import { DataTable, DataTableColumnHeader } from "@/components/ui/data-table"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
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
import { useAuth } from "@clerk/nextjs"
import { Policy } from "@/lib/types"
import { formatDate } from "@/lib/utils"
import {
  Plus,
  Shield,
  AlertTriangle,
  MoreHorizontal,
  Eye,
  Pencil,
  Trash2,
  Download,
  LayoutGrid,
  LayoutList,
  RefreshCw,
} from "lucide-react"
import Link from "next/link"
import { useToast } from "@/hooks/use-toast"

const policyTypeLabels: Record<string, string> = {
  max_spend: "Max Spend",
  block_instance_type: "Block Instance Type",
  auto_stop_idle: "Auto Stop Idle",
  require_tags: "Require Tags",
}

const policyTypeVariants: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  max_spend: "default",
  block_instance_type: "destructive",
  auto_stop_idle: "secondary",
  require_tags: "outline",
}

export default function PoliciesPage() {
  const { getToken } = useAuth()
  const queryClient = useQueryClient()
  const { toast } = useToast()
  const [view, setView] = useState<"table" | "cards">("table")
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [policyToDelete, setPolicyToDelete] = useState<Policy | null>(null)

  const { data: policies, isLoading, refetch } = useQuery<Policy[]>({
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

  const deletePolicy = useMutation({
    mutationFn: async (id: string) => {
      const token = await getToken()
      if (!token) throw new Error("Not authenticated")
      return apiRequestWithAuth(`/api/policies/${id}`, token, {
        method: "DELETE",
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["policies"] })
      toast({
        title: "Policy deleted",
        description: "The policy has been deleted.",
      })
      setDeleteDialogOpen(false)
      setPolicyToDelete(null)
    },
    onError: () => {
      toast({
        title: "Error",
        description: "Failed to delete policy",
        variant: "destructive",
      })
    },
  })

  const exportToCSV = () => {
    if (!policies?.length) return

    const headers = ["Name", "Description", "Type", "Enabled", "Violations", "Created At", "Updated At"]
    const rows = policies.map((p) => [
      `"${p.name.replace(/"/g, '""')}"`,
      `"${p.description.replace(/"/g, '""')}"`,
      p.type,
      p.enabled ? "Yes" : "No",
      p.violations?.length || 0,
      p.createdAt,
      p.updatedAt,
    ])

    const csv = [headers.join(","), ...rows.map((r) => r.join(","))].join("\n")
    const blob = new Blob([csv], { type: "text/csv" })
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = `policies-${new Date().toISOString().split("T")[0]}.csv`
    a.click()
    window.URL.revokeObjectURL(url)

    toast({
      title: "Export complete",
      description: "Policies have been exported to CSV",
    })
  }

  const columns: ColumnDef<Policy>[] = useMemo(
    () => [
      {
        accessorKey: "enabled",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Status" />
        ),
        cell: ({ row }) => {
          const policy = row.original
          return (
            <Switch
              checked={policy.enabled}
              onCheckedChange={(enabled) =>
                togglePolicy.mutate({ id: policy.id, enabled })
              }
            />
          )
        },
        filterFn: (row, id, value) => {
          if (value === "enabled") return row.getValue(id) === true
          if (value === "disabled") return row.getValue(id) === false
          return true
        },
      },
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Name" />
        ),
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            <Shield className="h-4 w-4 text-primary" />
            <span className="font-medium">{row.getValue("name")}</span>
          </div>
        ),
      },
      {
        accessorKey: "type",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Type" />
        ),
        cell: ({ row }) => {
          const type = row.getValue("type") as string
          return (
            <Badge variant={policyTypeVariants[type] || "default"}>
              {policyTypeLabels[type] || type.replace(/_/g, " ")}
            </Badge>
          )
        },
        filterFn: (row, id, value) => value === row.getValue(id),
      },
      {
        accessorKey: "description",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Description" />
        ),
        cell: ({ row }) => (
          <div className="max-w-[250px] truncate" title={row.getValue("description")}>
            {row.getValue("description")}
          </div>
        ),
      },
      {
        accessorKey: "violations",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Violations" />
        ),
        cell: ({ row }) => {
          const violations = row.original.violations || []
          const pending = violations.filter((v) => v.status === "pending").length
          if (pending > 0) {
            return (
              <Badge variant="destructive" className="gap-1">
                <AlertTriangle className="h-3 w-3" />
                {pending}
              </Badge>
            )
          }
          return <span className="text-muted-foreground">0</span>
        },
      },
      {
        accessorKey: "updatedAt",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Updated" />
        ),
        cell: ({ row }) => formatDate(row.getValue("updatedAt")),
      },
      {
        id: "actions",
        cell: ({ row }) => {
          const policy = row.original
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
                <DropdownMenuItem asChild>
                  <Link href={`/dashboard/policies/${policy.id}`}>
                    <Eye className="mr-2 h-4 w-4" />
                    View details
                  </Link>
                </DropdownMenuItem>
                <DropdownMenuItem asChild>
                  <Link href={`/dashboard/policies/${policy.id}/edit`}>
                    <Pencil className="mr-2 h-4 w-4" />
                    Edit
                  </Link>
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  className="text-destructive focus:text-destructive"
                  onClick={() => {
                    setPolicyToDelete(policy)
                    setDeleteDialogOpen(true)
                  }}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )
        },
      },
    ],
    [togglePolicy]
  )

  // Stats
  const stats = useMemo(() => {
    if (!policies?.length) return { total: 0, enabled: 0, withViolations: 0 }
    return {
      total: policies.length,
      enabled: policies.filter((p) => p.enabled).length,
      withViolations: policies.filter(
        (p) => p.violations?.some((v) => v.status === "pending")
      ).length,
    }
  }, [policies])

  if (isLoading) {
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
          <h1 className="text-3xl font-bold">Policies</h1>
          <p className="text-muted-foreground">Manage your governance policies</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
          <Link href="/dashboard/policies/new">
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              New Policy
            </Button>
          </Link>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Policies</CardTitle>
            <Shield className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.total}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Policies</CardTitle>
            <Shield className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-500">{stats.enabled}</div>
            <p className="text-xs text-muted-foreground">
              {stats.total > 0
                ? `${Math.round((stats.enabled / stats.total) * 100)}% of total`
                : "No policies"}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">With Violations</CardTitle>
            <AlertTriangle className="h-4 w-4 text-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-500">{stats.withViolations}</div>
            <p className="text-xs text-muted-foreground">Require attention</p>
          </CardContent>
        </Card>
      </div>

      {/* View Toggle and Table/Cards */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>All Policies</CardTitle>
              <CardDescription>
                A list of all governance policies in your organization
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <Tabs value={view} onValueChange={(v) => setView(v as "table" | "cards")}>
                <TabsList>
                  <TabsTrigger value="table">
                    <LayoutList className="h-4 w-4" />
                  </TabsTrigger>
                  <TabsTrigger value="cards">
                    <LayoutGrid className="h-4 w-4" />
                  </TabsTrigger>
                </TabsList>
              </Tabs>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {view === "table" ? (
            <DataTable
              columns={columns}
              data={policies || []}
              searchPlaceholder="Search policies..."
              filterableColumns={[
                {
                  id: "type",
                  title: "Type",
                  options: [
                    { label: "Max Spend", value: "max_spend" },
                    { label: "Block Instance Type", value: "block_instance_type" },
                    { label: "Auto Stop Idle", value: "auto_stop_idle" },
                    { label: "Require Tags", value: "require_tags" },
                  ],
                },
                {
                  id: "enabled",
                  title: "Status",
                  options: [
                    { label: "Enabled", value: "enabled" },
                    { label: "Disabled", value: "disabled" },
                  ],
                },
              ]}
              onExport={exportToCSV}
              exportLabel="Export CSV"
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {policies?.map((policy) => (
                <Card key={policy.id} className="relative">
                  <CardHeader>
                    <div className="flex items-center justify-between">
                      <Shield className="h-5 w-5 text-primary" />
                      <Switch
                        checked={policy.enabled}
                        onCheckedChange={(enabled) =>
                          togglePolicy.mutate({ id: policy.id, enabled })
                        }
                      />
                    </div>
                    <CardTitle className="text-lg">{policy.name}</CardTitle>
                    <CardDescription className="line-clamp-2">
                      {policy.description}
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-3">
                      <div className="flex items-center justify-between">
                        <Badge variant={policyTypeVariants[policy.type] || "default"}>
                          {policyTypeLabels[policy.type] || policy.type.replace(/_/g, " ")}
                        </Badge>
                        {policy.violations && policy.violations.filter((v) => v.status === "pending").length > 0 && (
                          <Badge variant="destructive" className="gap-1">
                            <AlertTriangle className="h-3 w-3" />
                            {policy.violations.filter((v) => v.status === "pending").length} violations
                          </Badge>
                        )}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        Updated {formatDate(policy.updatedAt)}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}

          {(!policies || policies.length === 0) && (
            <div className="flex flex-col items-center justify-center py-12">
              <Shield className="h-12 w-12 text-muted-foreground mb-4" />
              <p className="text-muted-foreground mb-4">No policies yet</p>
              <Link href="/dashboard/policies/new">
                <Button>Create Your First Policy</Button>
              </Link>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Policy</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete the policy "{policyToDelete?.name}"? This
              action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => policyToDelete && deletePolicy.mutate(policyToDelete.id)}
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
