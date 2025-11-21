"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { apiRequestWithAuth } from "@/lib/api"
import { useAuth } from "@clerk/nextjs"
import { useToast } from "@/hooks/use-toast"
import { PolicyBuilder } from "@/components/policies/policy-builder"

const policySchema = z.object({
  name: z.string().min(1, "Name is required"),
  description: z.string().min(1, "Description is required"),
  type: z.enum(["max_spend", "block_instance_type", "auto_stop_idle", "require_tags"]),
})

type PolicyForm = z.infer<typeof policySchema>

export default function NewPolicyPage() {
  const router = useRouter()
  const { getToken } = useAuth()
  const { toast } = useToast()
  const [config, setConfig] = useState<Record<string, any>>({})

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<PolicyForm>({
    resolver: zodResolver(policySchema),
  })

  const policyType = watch("type")

  const onSubmit = async (data: PolicyForm) => {
    try {
      const token = await getToken()
      if (!token) throw new Error("Not authenticated")

      await apiRequestWithAuth("/api/policies", token, {
        method: "POST",
        body: JSON.stringify({
          ...data,
          config,
        }),
      })

      toast({
        title: "Policy created",
        description: "Your policy has been created successfully.",
      })

      router.push("/dashboard/policies")
    } catch (error) {
      toast({
        title: "Error",
        description: error instanceof Error ? error.message : "Failed to create policy",
        variant: "destructive",
      })
    }
  }

  return (
    <div className="space-y-8 max-w-4xl">
      <div>
        <h1 className="text-3xl font-bold">Create New Policy</h1>
        <p className="text-muted-foreground">Build a governance policy with our no-code builder</p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Basic Information</CardTitle>
            <CardDescription>Provide a name and description for your policy</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Policy Name</Label>
              <Input
                id="name"
                {...register("name")}
                placeholder="e.g., Max Monthly Spend - Production"
              />
              {errors.name && (
                <p className="text-sm text-destructive">{errors.name.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Input
                id="description"
                {...register("description")}
                placeholder="Describe what this policy does"
              />
              {errors.description && (
                <p className="text-sm text-destructive">{errors.description.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="type">Policy Type</Label>
              <Select
                onValueChange={(value) => setValue("type", value as PolicyForm["type"])}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select a policy type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="max_spend">Max Monthly Spend</SelectItem>
                  <SelectItem value="block_instance_type">Block Instance Type</SelectItem>
                  <SelectItem value="auto_stop_idle">Auto-Stop Idle Resources</SelectItem>
                  <SelectItem value="require_tags">Require Mandatory Tags</SelectItem>
                </SelectContent>
              </Select>
              {errors.type && (
                <p className="text-sm text-destructive">{errors.type.message}</p>
              )}
            </div>
          </CardContent>
        </Card>

        {policyType && (
          <Card>
            <CardHeader>
              <CardTitle>Policy Configuration</CardTitle>
              <CardDescription>Configure the parameters for your policy</CardDescription>
            </CardHeader>
            <CardContent>
              <PolicyBuilder type={policyType} config={config} onChange={setConfig} />
            </CardContent>
          </Card>
        )}

        <div className="flex gap-4">
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Creating..." : "Create Policy"}
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={() => router.back()}
          >
            Cancel
          </Button>
        </div>
      </form>
    </div>
  )
}

