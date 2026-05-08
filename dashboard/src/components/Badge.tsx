"use client";

import { type ReactNode } from "react";

type BadgeVariant =
  | "default"
  | "success"
  | "warning"
  | "danger"
  | "info"
  | "purple"
  | "outline";

const variantClasses: Record<BadgeVariant, string> = {
  default: "bg-zinc-700/50 text-zinc-300",
  success: "bg-emerald-500/15 text-emerald-400 border border-emerald-500/20",
  warning: "bg-yellow-500/15 text-yellow-400 border border-yellow-500/20",
  danger: "bg-red-500/15 text-red-400 border border-red-500/20",
  info: "bg-blue-500/15 text-blue-400 border border-blue-500/20",
  purple: "bg-purple-500/15 text-purple-400 border border-purple-500/20",
  outline: "bg-transparent text-zinc-400 border border-zinc-700",
};

interface BadgeProps {
  variant?: BadgeVariant;
  children: ReactNode;
  className?: string;
  dot?: boolean;
}

export function Badge({
  variant = "default",
  children,
  className = "",
  dot = false,
}: BadgeProps) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${variantClasses[variant]} ${className}`}
    >
      {dot && (
        <span
          className={`h-1.5 w-1.5 rounded-full ${
            variant === "success"
              ? "bg-emerald-400"
              : variant === "warning"
              ? "bg-yellow-400"
              : variant === "danger"
              ? "bg-red-400"
              : variant === "info"
              ? "bg-blue-400"
              : variant === "purple"
              ? "bg-purple-400"
              : "bg-zinc-400"
          }`}
        />
      )}
      {children}
    </span>
  );
}

export default Badge;
