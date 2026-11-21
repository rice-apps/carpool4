"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "../lib/auth";
import { Button } from "./ui/button";

const navigation = [
  { href: "/", label: "Find a ride" },
  { href: "/my-rides", label: "My rides" },
  { href: "/rides/new", label: "Post a ride" },
  { href: "/profile", label: "Profile" },
];

export function Brand() {
  return (
    <Link
      href="/"
      className="text-lg font-semibold tracking-tight"
      aria-label="Carpool home"
    >
      Carpool
    </Link>
  );
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { session } = useAuth();

  return (
    <>
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:block focus:p-3"
      >
        Skip to content
      </a>
      <header className="border-b bg-card">
        <div className="page-container flex flex-wrap items-center justify-between gap-3 py-4">
          <Brand />
          <nav className="flex flex-wrap gap-1" aria-label="Main navigation">
            {navigation.map(({ href, label }) => (
              <Button
                key={href}
                variant={pathname === href ? "secondary" : "ghost"}
                size="sm"
                asChild
              >
                <Link
                  href={href}
                  aria-current={pathname === href ? "page" : undefined}
                >
                  {label}
                </Link>
              </Button>
            ))}
            {!session && (
              <Button variant="outline" size="sm" asChild>
                <Link href="/login">Sign in</Link>
              </Button>
            )}
          </nav>
        </div>
      </header>
      <main id="main-content">{children}</main>
    </>
  );
}
