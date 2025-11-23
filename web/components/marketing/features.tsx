import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollText, Cloud, Activity, Webhook, Shield, TrendingUp } from "lucide-react";

const features = [
  {
    icon: <ScrollText className="w-6 h-6 text-primary" />,
    title: "No-Code Policy Builder",
    description: "Create governance policies with a visual drag-and-drop interface. No Rego knowledge required.",
  },
  {
    icon: <Cloud className="w-6 h-6 text-primary" />,
    title: "Multi-Cloud Support",
    description: "Connect AWS, Azure, and GCP. Manage all your cloud spend from one dashboard.",
  },
  {
    icon: <Shield className="w-6 h-6 text-primary" />,
    title: "Open Policy Agent",
    description: "Powered by OPA for enterprise-grade policy evaluation and enforcement.",
  },
  {
    icon: <Activity className="w-6 h-6 text-primary" />,
    title: "Real-Time Monitoring",
    description: "Track spending, violations, and remediations in real-time with beautiful charts.",
  },
  {
    icon: <Webhook className="w-6 h-6 text-primary" />,
    title: "Webhook Integrations",
    description: "Get notified via Slack, Discord, or Microsoft Teams when policies are violated.",
  },
  {
    icon: <TrendingUp className="w-6 h-6 text-primary" />,
    title: "Cost Analytics",
    description: "Deep insights into your cloud spending patterns with trend analysis.",
  },
];

export function Features() {
  return (
    <section className="py-20 bg-muted/50">
      <div className="container mx-auto">
        <h2 className="text-3xl font-bold text-center mb-12">Enterprise-Grade Features</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {features.map((feature, index) => (
            <Card key={index} className="hover:shadow-lg transition-shadow duration-300">
              <CardHeader>
                {feature.icon}
                <CardTitle>{feature.title}</CardTitle>
              </CardHeader>
              <CardContent>
                <CardDescription>{feature.description}</CardDescription>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    </section>
  );
}

