"use client"

import { Button } from "@/components/ui/button"
import { SignedIn, SignedOut, SignUpButton } from "@clerk/nextjs"
import Link from "next/link"
import { ArrowRight, Shield, Zap, TrendingDown, CheckCircle2 } from "lucide-react"

export function Hero() {
  return (
    <section className="relative overflow-hidden">
      {/* Background gradient */}
      <div className="absolute inset-0 -z-10 bg-[radial-gradient(ellipse_80%_80%_at_50%_-20%,rgba(59,130,246,0.1),transparent)]" />

      <div className="container py-20 md:py-32 px-4 md:px-6">
        <div className="flex flex-col items-center text-center space-y-8">
          {/* Badge */}
          <div className="animate-fade-in inline-flex items-center gap-2 rounded-full border bg-muted/50 px-4 py-1.5 text-sm">
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
              <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500" />
            </span>
            <span className="text-muted-foreground">Now with AI-powered cost predictions</span>
          </div>

          {/* Main heading */}
          <div className="space-y-4 animate-fade-in animation-delay-100">
            <h1 className="text-4xl md:text-5xl lg:text-6xl font-bold tracking-tight max-w-4xl">
              Take Control of Your
              <span className="gradient-text"> Cloud Spend</span>
            </h1>
            <p className="text-lg md:text-xl text-muted-foreground max-w-2xl mx-auto leading-relaxed">
              Policy-governance-first platform that automatically enforces spending rules,
              prevents budget overruns, and remediates violations in real-time.
            </p>
          </div>

          {/* CTA Buttons */}
          <div className="flex flex-col sm:flex-row gap-4 animate-fade-in animation-delay-200">
            <SignedOut>
              <SignUpButton mode="modal">
                <Button size="lg" className="text-base px-8 h-12 font-medium group">
                  Get Started Free
                  <ArrowRight className="ml-2 h-4 w-4 transition-transform group-hover:translate-x-1" />
                </Button>
              </SignUpButton>
            </SignedOut>
            <SignedIn>
              <Link href="/dashboard">
                <Button size="lg" className="text-base px-8 h-12 font-medium group">
                  Go to Dashboard
                  <ArrowRight className="ml-2 h-4 w-4 transition-transform group-hover:translate-x-1" />
                </Button>
              </Link>
            </SignedIn>
            <Link href="#features">
              <Button size="lg" variant="outline" className="text-base px-8 h-12 font-medium">
                Learn More
              </Button>
            </Link>
          </div>

          {/* Trust indicators */}
          <div className="flex flex-wrap justify-center gap-x-8 gap-y-2 text-sm text-muted-foreground animate-fade-in animation-delay-300">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-500" />
              <span>No credit card required</span>
            </div>
            <div className="flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-500" />
              <span>14-day free trial</span>
            </div>
            <div className="flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-500" />
              <span>Cancel anytime</span>
            </div>
          </div>

          {/* Feature cards */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mt-16 w-full max-w-4xl">
            <div className="enterprise-card flex flex-col items-center space-y-3 p-6 animate-fade-in animation-delay-300">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10">
                <Shield className="h-6 w-6 text-primary" />
              </div>
              <h3 className="font-semibold">Policy-First</h3>
              <p className="text-sm text-muted-foreground text-center leading-relaxed">
                Define governance rules with our intuitive no-code policy builder
              </p>
            </div>
            <div className="enterprise-card flex flex-col items-center space-y-3 p-6 animate-fade-in animation-delay-400">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10">
                <Zap className="h-6 w-6 text-primary" />
              </div>
              <h3 className="font-semibold">Auto-Remediate</h3>
              <p className="text-sm text-muted-foreground text-center leading-relaxed">
                Automatically stop or terminate resources that violate policies
              </p>
            </div>
            <div className="enterprise-card flex flex-col items-center space-y-3 p-6 animate-fade-in animation-delay-500">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10">
                <TrendingDown className="h-6 w-6 text-primary" />
              </div>
              <h3 className="font-semibold">Save Money</h3>
              <p className="text-sm text-muted-foreground text-center leading-relaxed">
                Reduce cloud waste and prevent unexpected bills automatically
              </p>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
