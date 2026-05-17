"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Flag,
  FlaskConical,
  Globe,
  History,
  Settings,
  LayoutDashboard,
  LogIn,
  LogOut,
} from "lucide-react";
import { useAuth } from "@/lib/auth-context";

const navItems = [
  { href: "/", label: "Dashboard", icon: LayoutDashboard },
  { href: "/flags", label: "Flags", icon: Flag },
  { href: "/experiments", label: "Experiments", icon: FlaskConical },
  { href: "/environments", label: "Environments", icon: Globe },
  { href: "/audit", label: "Audit Log", icon: History },
  { href: "/settings", label: "Settings", icon: Settings },
];

export default function Sidebar() {
  const pathname = usePathname();
  const { user, signOut } = useAuth();

  function isActive(href: string) {
    if (href === "/") return pathname === "/";
    return pathname.startsWith(href);
  }

  return (
    <aside className="fixed left-0 top-0 z-40 flex h-screen w-60 flex-col border-r border-zinc-800 bg-zinc-950">
      {/* Logo */}
      <div className="flex h-16 items-center gap-2.5 border-b border-zinc-800 px-5">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-600">
          <Flag className="h-4 w-4 text-white" />
        </div>
        <span className="text-lg font-semibold text-white tracking-tight">
          Rollout
        </span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto px-3 py-4">
        <ul className="flex flex-col gap-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const active = isActive(item.href);
            return (
              <li key={item.href}>
                <Link
                  href={item.href}
                  className={`flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                    active
                      ? "bg-zinc-800 text-white"
                      : "text-zinc-400 hover:bg-zinc-900 hover:text-zinc-100"
                  }`}
                >
                  <Icon className="h-4.5 w-4.5 shrink-0" />
                  {item.label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      {/* Footer */}
      <div className="border-t border-zinc-800 px-4 py-4">
        {user ? (
          <div className="space-y-3">
            <div>
              <p className="truncate text-sm font-medium text-zinc-200">
                {user.user_metadata?.name ?? user.email}
              </p>
              <p className="truncate text-xs text-zinc-500">{user.email}</p>
            </div>
            <button
              type="button"
              onClick={() => void signOut()}
              className="flex w-full items-center gap-2 rounded-lg px-2 py-2 text-sm text-zinc-400 transition-colors hover:bg-zinc-900 hover:text-zinc-100"
            >
              <LogOut className="h-4 w-4" />
              Sign out
            </button>
          </div>
        ) : (
          <div className="space-y-3">
            <div>
              <p className="text-sm font-medium text-zinc-200">Public demo</p>
              <p className="text-xs text-zinc-500">Read-only sample workspace</p>
            </div>
            <Link
              href="/login"
              className="flex w-full items-center gap-2 rounded-lg bg-blue-600 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-500"
            >
              <LogIn className="h-4 w-4" />
              Sign in
            </Link>
          </div>
        )}
        <p className="mt-4 text-xs text-zinc-600">Rollout v1.0.0</p>
      </div>
    </aside>
  );
}
