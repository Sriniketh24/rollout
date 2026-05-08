"use client";

import { useState } from "react";
import { ShieldOff, X, AlertTriangle } from "lucide-react";

interface KillSwitchButtonProps {
  flagName: string;
  environmentName: string;
  isActive: boolean;
  onActivate: () => void;
  onDeactivate: () => void;
}

export function KillSwitchButton({
  flagName,
  environmentName,
  isActive,
  onActivate,
  onDeactivate,
}: KillSwitchButtonProps) {
  const [showModal, setShowModal] = useState(false);
  const [confirmText, setConfirmText] = useState("");

  const requiredText = "KILL";

  function handleConfirm() {
    if (isActive) {
      onDeactivate();
    } else {
      onActivate();
    }
    setShowModal(false);
    setConfirmText("");
  }

  return (
    <>
      <button
        onClick={() => setShowModal(true)}
        className={`flex items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium transition-all ${
          isActive
            ? "bg-red-500/20 text-red-400 border border-red-500/30 hover:bg-red-500/30"
            : "bg-zinc-800 text-zinc-300 border border-zinc-700 hover:bg-red-500/10 hover:text-red-400 hover:border-red-500/30"
        }`}
      >
        <ShieldOff className="h-4 w-4" />
        {isActive ? "Kill Switch Active" : "Kill Switch"}
      </button>

      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="w-full max-w-md rounded-xl border border-zinc-800 bg-zinc-900 p-6 shadow-2xl">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-red-500/15">
                  <AlertTriangle className="h-5 w-5 text-red-400" />
                </div>
                <h3 className="text-lg font-semibold text-zinc-100">
                  {isActive ? "Deactivate Kill Switch" : "Activate Kill Switch"}
                </h3>
              </div>
              <button
                onClick={() => {
                  setShowModal(false);
                  setConfirmText("");
                }}
                className="text-zinc-500 hover:text-zinc-300"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="mt-4 rounded-lg bg-red-500/10 border border-red-500/20 p-4">
              <p className="text-sm text-red-300">
                {isActive
                  ? `This will re-enable "${flagName}" in ${environmentName}. Traffic will resume receiving the flag value.`
                  : `This will immediately disable "${flagName}" in ${environmentName}. All users will receive the off variation.`}
              </p>
            </div>

            {!isActive && (
              <div className="mt-4">
                <label className="block text-sm text-zinc-400 mb-2">
                  Type <span className="font-mono font-bold text-red-400">KILL</span> to
                  confirm
                </label>
                <input
                  type="text"
                  value={confirmText}
                  onChange={(e) => setConfirmText(e.target.value)}
                  placeholder="KILL"
                  className="w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-red-500/50 focus:outline-none focus:ring-1 focus:ring-red-500/30"
                />
              </div>
            )}

            <div className="mt-6 flex justify-end gap-3">
              <button
                onClick={() => {
                  setShowModal(false);
                  setConfirmText("");
                }}
                className="rounded-lg px-4 py-2 text-sm font-medium text-zinc-400 hover:text-zinc-200 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleConfirm}
                disabled={!isActive && confirmText !== requiredText}
                className={`rounded-lg px-4 py-2 text-sm font-semibold transition-all ${
                  isActive
                    ? "bg-emerald-600 text-white hover:bg-emerald-500"
                    : "bg-red-600 text-white hover:bg-red-500 disabled:opacity-40 disabled:cursor-not-allowed"
                }`}
              >
                {isActive ? "Deactivate" : "Activate Kill Switch"}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

export default KillSwitchButton;
