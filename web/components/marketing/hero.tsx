"use client"

import { Button } from "@/components/ui/button"
import { SignedOut, SignUpButton } from "@clerk/nextjs"
import Link from "next/link"
import { ArrowRight, Shield, Zap, TrendingDown } from "lucide-react"

export function Hero() {
  return (
    <section className="container py-20 md:py-32">
      <div className="flex flex-col items-center text-center space-y-8">
        <div className="space-y-4">
          <h1 className="text-4xl md:text-6xl font-bold tracking-tight">
            Take Control of Your
            <span className="text-primary"> Cloud Spend</span>
          </h1>
          <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
            Policy-governance-first platform that automatically enforces spending rules,
            prevents budget overruns, and remediates violations in real-time.
          </p>
        </div>
        <div className="flex flex-col sm:flex-row gap-4">
          <SignedOut>
            <SignUpButton mode="modal">
              <Button size="lg" className="text-lg px-8">
                Get Started Free
                <ArrowRight className="ml-2 h-5 w-5" />
              </Button>
            </SignUpButton>
          </SignedOut>
          <SignedOut>
            <Link href="/dashboard">
              <Button size="lg" variant="outline" className="text-lg px-8">
                View Dashboard
              </Button>
            </Link>
          </SignedOut>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8 mt-16 w-full max-w-4xl">
          <div className="flex flex-col items-center space-y-2 p-6 rounded-lg border">
            <Shield className="h-10 w-10 text-primary mb-2" />
            <h3 className="font-semibold">Policy-First</h3>
            <p className="text-sm text-muted-foreground text-center">
              Define governance rules with our no-code policy builder
            </p>
          </div>
          <div className="flex flex-col items-center space-y-2 p-6 rounded-lg border">
            <Zap className="h-10 w-10 text-primary mb-2" />
            <h3 className="font-semibold">Auto-Remediate</h3>
            <p className="text-sm text-muted-foreground text-center">
              Automatically stop or terminate resources that violate policies
            </p>
          </div>
          <div className="flex flex-col items-center space-y-2 p-6 rounded-lg border">
            <TrendingDown className="h-10 w-10 text-primary mb-2" />
            <h3 className="font-semibold">Save Money</h3>
            <p className="text-sm text-muted-foreground text-center">
              Reduce cloud waste and prevent unexpected bills
            </p>
          </div>
        </div>
      </div>
    </section>
  )
}

