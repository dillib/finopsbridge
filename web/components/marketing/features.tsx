import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Policy, Cloud, Activity, Webhook, Shield, TrendingUp } from "lucide-react"

const features = [
  {
    icon: Policy,
    title: "No-Code Policy Builder",
    description: "Create governance policies with a visual drag-and-drop interface. No Rego knowledge required.",
  },
  {
    icon: Cloud,
    title: "Multi-Cloud Support",
    description: "Connect AWS, Azure, and GCP. Manage all your cloud spend from one dashboard.",
  },
  {
    icon: Shield,
    title: "Open Policy Agent",
    description: "Powered by OPA for enterprise-grade policy evaluation and enforcement.",
  },
  {
    icon: Activity,
    title: "Real-Time Monitoring",
    description: "Track spending, violations, and remediations in real-time with beautiful charts.",
  },
  {
    icon: Webhook,
    title: "Webhook Integrations",
    description: "Get notified via Slack, Discord, or Microsoft Teams when policies are violated.",
  },
  {
    icon: TrendingUp,
    title: "Cost Analytics",
    description: "Deep insights into your cloud spending patterns with trend analysis.",
  },
]

export function Features() {
  return (
    <section className="container py-20">
      <div className="text-center space-y-4 mb-12">
        <h2 className="text-3xl md:text-4xl font-bold">Everything You Need</h2>
        <p className="text-muted-foreground max-w-2xl mx-auto">
          A complete platform for cloud spend governance and control
        </p>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {features.map((feature) => {
          const Icon = feature.icon
          return (
            <Card key={feature.title}>
              <CardHeader>
                <Icon className="h-8 w-8 text-primary mb-2" />
                <CardTitle>{feature.title}</CardTitle>
                <CardDescription>{feature.description}</CardDescription>
              </CardHeader>
            </Card>
          )
        })}
      </div>
    </section>
  )
}

