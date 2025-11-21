"use client"

import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

interface PolicyBuilderProps {
  type: string
  config: Record<string, any>
  onChange: (config: Record<string, any>) => void
}

export function PolicyBuilder({ type, config, onChange }: PolicyBuilderProps) {
  const updateConfig = (key: string, value: any) => {
    onChange({ ...config, [key]: value })
  }

  switch (type) {
    case "max_spend":
      return (
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="maxAmount">Maximum Monthly Spend (USD)</Label>
            <Input
              id="maxAmount"
              type="number"
              value={config.maxAmount || ""}
              onChange={(e) => updateConfig("maxAmount", parseFloat(e.target.value))}
              placeholder="1000"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="accountId">Account/Project ID (optional)</Label>
            <Input
              id="accountId"
              value={config.accountId || ""}
              onChange={(e) => updateConfig("accountId", e.target.value)}
              placeholder="Leave empty for all accounts"
            />
          </div>
        </div>
      )

    case "block_instance_type":
      return (
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="maxSize">Maximum Instance Size</Label>
            <Select
              value={config.maxSize || ""}
              onValueChange={(value) => updateConfig("maxSize", value)}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select maximum size" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="small">Small</SelectItem>
                <SelectItem value="medium">Medium</SelectItem>
                <SelectItem value="large">Large</SelectItem>
                <SelectItem value="xlarge">X-Large</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      )

    case "auto_stop_idle":
      return (
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="idleHours">Idle Time (hours)</Label>
            <Input
              id="idleHours"
              type="number"
              value={config.idleHours || ""}
              onChange={(e) => updateConfig("idleHours", parseInt(e.target.value))}
              placeholder="24"
            />
          </div>
        </div>
      )

    case "require_tags":
      return (
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="requiredTags">Required Tags (comma-separated)</Label>
            <Input
              id="requiredTags"
              value={config.requiredTags || ""}
              onChange={(e) => updateConfig("requiredTags", e.target.value.split(",").map(t => t.trim()))}
              placeholder="Environment, Team, Project"
            />
          </div>
        </div>
      )

    default:
      return <p className="text-muted-foreground">Select a policy type to configure</p>
  }
}

