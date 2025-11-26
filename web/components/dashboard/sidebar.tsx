"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { cn } from "@/lib/utils"
import { LayoutDashboard, Shield, Activity, Settings, Cloud, Building2, AlertTriangle, Library, Sparkles } from "lucide-react"
import { OrganizationSwitcher, UserButton } from "@clerk/nextjs"
import { Suspense } from "react"

const navItems = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/dashboard/policies", label: "Policies", icon: Shield },
  { href: "/dashboard/policy-library", label: "Policy Library", icon: Library },
  { href: "/dashboard/recommendations", label: "AI Recommendations", icon: Sparkles },
  { href: "/dashboard/violations", label: "Violations", icon: AlertTriangle },
  { href: "/dashboard/clouds", label: "Cloud Providers", icon: Cloud },
  { href: "/dashboard/activity", label: "Activity Log", icon: Activity },
  { href: "/dashboard/settings", label: "Settings", icon: Settings },
]

function OrganizationSwitcherWrapper() {
  return (
    <OrganizationSwitcher
      createOrganizationMode="modal"
      afterCreateOrganizationUrl="/dashboard"
      afterSelectOrganizationUrl="/dashboard"
      appearance={{
        elements: {
          rootBox: "w-full",
          organizationSwitcherTrigger: "w-full justify-between"
        }
      }}
    />
  )
}

function OrganizationSwitcherFallback() {
  return (
    <div className="flex items-center space-x-2 text-sm text-muted-foreground">
      <Building2 className="h-4 w-4" />
      <span>Loading...</span>
    </div>
  )
}

export function Sidebar() {
  const pathname = usePathname()

  return (
    <aside className="w-64 border-r bg-background p-6 flex flex-col">
      <div className="mb-4">
        <h2 className="text-xl font-bold">FinOpsBridge</h2>
      </div>
      <div className="mb-6">
        <Suspense fallback={<OrganizationSwitcherFallback />}>
          <OrganizationSwitcherWrapper />
        </Suspense>
      </div>
      <nav className="space-y-2">
        {navItems.map((item) => {
          const Icon = item.icon
          const isActive = pathname === item.href || pathname?.startsWith(item.href + "/")
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center space-x-3 px-4 py-2 rounded-lg transition-colors",
                isActive
                  ? "bg-primary text-primary-foreground"
                  : "hover:bg-accent hover:text-accent-foreground"
              )}
            >
              <Icon className="h-5 w-5" />
              <span>{item.label}</span>
            </Link>
          )
        })}
      </nav>
      <div className="mt-auto pt-6 border-t">
        <UserButton
          afterSignOutUrl="/"
          appearance={{
            elements: {
              rootBox: "w-full",
              userButtonTrigger: "w-full justify-start"
            }
          }}
        />
      </div>
    </aside>
  )
}

