"use client";

import { useState } from "react";
import {
  Key,
  Copy,
  Check,
  RefreshCw,
  Users,
  Eye,
  EyeOff,
  Crown,
  Shield,
  Pencil,
  User,
  Settings as SettingsIcon,
} from "lucide-react";
import { Badge } from "@/components/Badge";
import type { UserRole } from "@/lib/types";
import { useAuth } from "@/lib/auth-context";
import { useDemoData } from "@/lib/use-demo-data";

export default function SettingsPage() {
  const { user } = useAuth();
  const { data, isDemo } = useDemoData();
  const [showApiKey, setShowApiKey] = useState(false);
  const [showServerKey, setShowServerKey] = useState(false);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  function copyToClipboard(text: string, keyId: string) {
    navigator.clipboard.writeText(text);
    setCopiedKey(keyId);
    setTimeout(() => setCopiedKey(null), 2000);
  }

  function maskKey(key: string) {
    return key.slice(0, 12) + "*".repeat(key.length - 16) + key.slice(-4);
  }

  const roleIcons: Record<UserRole, React.ElementType> = {
    owner: Crown,
    admin: Shield,
    editor: Pencil,
    viewer: Eye,
  };

  const roleVariants: Record<UserRole, "warning" | "info" | "purple" | "default"> = {
    owner: "warning",
    admin: "info",
    editor: "purple",
    viewer: "default",
  };

  const project = data.project;
  const users = user
    ? [
        {
          id: user.id,
          name: user.user_metadata?.name ?? user.email ?? "Workspace user",
          email: user.email ?? "",
          role: "owner" as UserRole,
          createdAt: user.created_at,
          lastLogin: user.last_sign_in_at,
        },
      ]
    : [
        {
          id: "demo-admin",
          name: "Demo Admin",
          email: "demo@example.com",
          role: "owner" as UserRole,
          createdAt: project.createdAt,
          lastLogin: project.updatedAt,
        },
      ];

  return (
    <div className="space-y-8 max-w-4xl">
      <div>
        <h1 className="text-2xl font-bold text-zinc-100">Settings</h1>
        <p className="mt-1 text-sm text-zinc-400">
          Manage your project configuration, API keys, and team
        </p>
      </div>

      {/* Project info */}
      <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6 space-y-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10">
            <SettingsIcon className="h-5 w-5 text-blue-400" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-zinc-100">Project</h2>
            <p className="text-sm text-zinc-400">
              {project.description}
              {isDemo ? " Public demo workspace." : ""}
            </p>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-zinc-300 mb-1.5">
              Project Name
            </label>
            <input
              type="text"
              defaultValue={project.name}
              className="w-full rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2.5 text-sm text-zinc-100 focus:border-blue-500/50 focus:outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-zinc-300 mb-1.5">
              Project Key
            </label>
            <input
              type="text"
              value={project.key}
              disabled
              className="w-full rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2.5 text-sm text-zinc-500 font-mono"
            />
          </div>
        </div>
      </div>

      {/* API keys */}
      <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6 space-y-5">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-yellow-500/10">
            <Key className="h-5 w-5 text-yellow-400" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-zinc-100">API Keys</h2>
            <p className="text-sm text-zinc-400">
              Use these keys to connect your application
            </p>
          </div>
        </div>

        {/* Client-side key */}
        <div className="rounded-lg border border-zinc-800 bg-zinc-950 p-4">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              <h3 className="text-sm font-medium text-zinc-200">
                Client-Side Key
              </h3>
              <Badge variant="success">public</Badge>
            </div>
          </div>
          <p className="text-xs text-zinc-500 mb-3">
            Safe to use in client-side code. Only allows flag evaluation.
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded-lg bg-zinc-900 border border-zinc-800 px-3 py-2 text-sm text-zinc-300 font-mono">
              {showApiKey ? project.apiKey : maskKey(project.apiKey)}
            </code>
            <button
              onClick={() => setShowApiKey(!showApiKey)}
              className="rounded-lg p-2 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800 transition-colors"
            >
              {showApiKey ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </button>
            <button
              onClick={() => copyToClipboard(project.apiKey, "client")}
              className="rounded-lg p-2 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800 transition-colors"
            >
              {copiedKey === "client" ? (
                <Check className="h-4 w-4 text-emerald-400" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </button>
          </div>
        </div>

        {/* Server-side key */}
        <div className="rounded-lg border border-zinc-800 bg-zinc-950 p-4">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              <h3 className="text-sm font-medium text-zinc-200">
                Server-Side Key
              </h3>
              <Badge variant="danger">secret</Badge>
            </div>
          </div>
          <p className="text-xs text-zinc-500 mb-3">
            Keep secret. Allows full API access including management operations.
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded-lg bg-zinc-900 border border-zinc-800 px-3 py-2 text-sm text-zinc-300 font-mono">
              {showServerKey
                ? project.serverApiKey
                : maskKey(project.serverApiKey)}
            </code>
            <button
              onClick={() => setShowServerKey(!showServerKey)}
              className="rounded-lg p-2 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800 transition-colors"
            >
              {showServerKey ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </button>
            <button
              onClick={() =>
                copyToClipboard(project.serverApiKey, "server")
              }
              className="rounded-lg p-2 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800 transition-colors"
            >
              {copiedKey === "server" ? (
                <Check className="h-4 w-4 text-emerald-400" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </button>
          </div>
        </div>

        <button className="flex items-center gap-2 rounded-lg border border-zinc-700 px-4 py-2 text-sm font-medium text-zinc-300 hover:bg-zinc-800 transition-colors">
          <RefreshCw className="h-4 w-4" />
          Regenerate Keys
        </button>
      </div>

      {/* Team members */}
      <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6 space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-500/10">
              <Users className="h-5 w-5 text-purple-400" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-zinc-100">
                Team Members
              </h2>
              <p className="text-sm text-zinc-400">
                {users.length} member{users.length !== 1 ? "s" : ""}
              </p>
            </div>
          </div>
          <button className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-500">
            Invite Member
          </button>
        </div>

        <div className="divide-y divide-zinc-800">
          {users.map((member) => {
            const RoleIcon = roleIcons[member.role];
            return (
              <div
                key={member.id}
                className="flex items-center justify-between py-4"
              >
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-full bg-zinc-800 text-zinc-400">
                    <User className="h-5 w-5" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-zinc-200">
                      {member.name}
                    </p>
                    <p className="text-xs text-zinc-500">{member.email}</p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <Badge variant={roleVariants[member.role]}>
                    <RoleIcon className="h-3 w-3 mr-1" />
                    {member.role}
                  </Badge>
                  {member.lastLogin && (
                    <span className="text-xs text-zinc-500">
                      Last login{" "}
                      {new Date(member.lastLogin).toLocaleDateString()}
                    </span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
