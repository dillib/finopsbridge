import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Check } from "lucide-react"
import { SignUpButton } from "@clerk/nextjs"

const plans = [
  {
    name: "Starter",
    price: "$99",
    description: "Perfect for small teams",
    features: [
      "Up to 3 cloud accounts",
      "10 active policies",
      "Basic reporting",
      "Email support",
      "Webhook integrations",
    ],
  },
  {
    name: "Professional",
    price: "$299",
    description: "For growing organizations",
    features: [
      "Unlimited cloud accounts",
      "Unlimited policies",
      "Advanced analytics",
      "Priority support",
      "Custom webhooks",
      "Activity logs",
    ],
    popular: true,
  },
  {
    name: "Enterprise",
    price: "Custom",
    description: "For large enterprises",
    features: [
      "Everything in Professional",
      "Dedicated support",
      "SLA guarantee",
      "Custom integrations",
      "On-premise deployment",
      "Advanced security",
    ],
  },
]

export function Pricing() {
  return (
    <section className="container py-20">
      <div className="text-center space-y-4 mb-12">
        <h2 className="text-3xl md:text-4xl font-bold">Simple Pricing</h2>
        <p className="text-muted-foreground max-w-2xl mx-auto">
          Choose the plan that fits your needs
        </p>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-5xl mx-auto">
        {plans.map((plan) => (
          <Card key={plan.name} className={plan.popular ? "border-primary" : ""}>
            <CardHeader>
              {plan.popular && (
                <div className="text-xs font-semibold text-primary mb-2">POPULAR</div>
              )}
              <CardTitle>{plan.name}</CardTitle>
              <CardDescription>{plan.description}</CardDescription>
              <div className="mt-4">
                <span className="text-4xl font-bold">{plan.price}</span>
                {plan.price !== "Custom" && <span className="text-muted-foreground">/month</span>}
              </div>
            </CardHeader>
            <CardContent>
              <ul className="space-y-3">
                {plan.features.map((feature) => (
                  <li key={feature} className="flex items-start">
                    <Check className="h-5 w-5 text-primary mr-2 flex-shrink-0 mt-0.5" />
                    <span className="text-sm">{feature}</span>
                  </li>
                ))}
              </ul>
            </CardContent>
            <CardFooter>
              <SignUpButton mode="modal">
                <Button className="w-full" variant={plan.popular ? "default" : "outline"}>
                  Get Started
                </Button>
              </SignUpButton>
            </CardFooter>
          </Card>
        ))}
      </div>
    </section>
  )
}

