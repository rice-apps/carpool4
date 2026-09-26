"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  CarProfile,
  Compass,
  Plus,
  UserCircle,
  ArrowUpRight,
} from "@phosphor-icons/react";
import { useAuth } from "../lib/auth";

const navigation = [
  { href: "/", label: "Find a ride", icon: Compass },
  { href: "/my-rides", label: "My rides", icon: CarProfile },
  { href: "/rides/new", label: "Post a ride", icon: Plus },
  { href: "/profile", label: "Profile", icon: UserCircle },
];

export function Brand() {
  return (
    <Link href="/" className="brand" aria-label="Carpool home">
      <span className="brand-mark" aria-hidden="true">
        <span />
        <span />
      </span>
      <span>carpool</span>
    </Link>
  );
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { session } = useAuth();
  const firstName = session?.user.user_metadata?.full_name?.split(" ")[0];

  return (
    <div className="app-frame">
      <header className="site-header">
        <div className="header-inner">
          <Brand />
          <nav className="desktop-nav" aria-label="Main navigation">
            {navigation.map(({ href, label }) => (
              <Link
                key={href}
                href={href}
                className={`nav-link ${pathname === href ? "is-active" : ""}`}
              >
                {label}
              </Link>
            ))}
          </nav>
          {session ? (
            <Link
              href="/profile"
              className="header-account"
              aria-label="Your profile"
            >
              <span>{firstName || "My account"}</span>
              <span className="header-avatar">{(firstName || "R")[0]}</span>
            </Link>
          ) : (
            <Link href="/login" className="header-account">
              Sign in
            </Link>
          )}
        </div>
      </header>
      <main id="main-content">{children}</main>
      <footer className="site-footer">
        <div className="page-container footer-inner">
          <span>Made for the ride from Rice to everywhere.</span>
          <Link href="/rides/new">
            Share your next trip <ArrowUpRight size={16} />
          </Link>
        </div>
      </footer>
      <nav className="mobile-nav" aria-label="Main navigation">
        {navigation.map(({ href, label, icon: Icon }) => (
          <Link
            key={href}
            href={href}
            className={pathname === href ? "is-active" : ""}
            aria-current={pathname === href ? "page" : undefined}
          >
            <Icon size={22} weight={pathname === href ? "fill" : "regular"} />
            <span>{label}</span>
          </Link>
        ))}
      </nav>
    </div>
  );
}
