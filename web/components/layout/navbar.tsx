"use client";

import { SignedIn, SignedOut, UserButton } from "@clerk/nextjs";
import { ModeToggle } from "@/components/ui/mode-toggle";
import Link from "next/link";
import { Button } from "@/components/ui/button";

export function Navbar() {
  return (
    <header className="sticky top-0 z-50 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto flex h-16 items-center justify-between px-4">
        <Link href="/" className="text-2xl font-bold text-primary">
          FinOpsBridge
        </Link>
        <nav className="flex items-center space-x-4">
          <Link href="/features" className="text-neutral hover:text-primary transition-colors">
            Features
          </Link>
          <Link href="/pricing" className="text-neutral hover:text-primary transition-colors">
            Pricing
          </Link>
          <SignedOut>
            <Link href="/sign-in">
              <Button variant="secondary">Sign In</Button>
            </Link>
          </SignedOut>
          <SignedIn>
            <Link href="/dashboard">
              <Button variant="default">Dashboard</Button>
            </Link>
            <UserButton afterSignOutUrl="/" />
          </SignedIn>
          <ModeToggle />
        </nav>
      </div>
    </header>
  );
}
