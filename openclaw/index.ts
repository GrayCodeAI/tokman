/**
 * TokMan Rewrite Plugin for OpenClaw
 *
 * Transparently rewrites exec tool commands to TokMan equivalents
 * before execution, achieving 60-90% LLM token savings.
 *
 * All rewrite logic lives in `tokman rewrite` (internal/discover/registry.go).
 * This plugin is a thin delegate — to add or change rules, edit the
 * Go registry, not this file.
 */

import { execSync, spawnSync } from "node:child_process";

let tokmanAvailable: boolean | null = null;

function checkTokman(): boolean {
  if (tokmanAvailable !== null) return tokmanAvailable;
  try {
    execSync("which tokman", { stdio: "ignore" });
    tokmanAvailable = true;
  } catch {
    tokmanAvailable = false;
  }
  return tokmanAvailable;
}

function tryRewrite(command: string): string | null {
  try {
    const result = spawnSync("tokman", ["rewrite", command], {
      encoding: "utf-8",
      timeout: 2000,
    });
    if (result.status !== 0 || result.error) return null;
    const out = result.stdout.trim();
    return out && out !== command ? out : null;
  } catch {
    return null;
  }
}

export default function register(api: any) {
  const pluginConfig = api.config ?? {};
  const enabled = pluginConfig.enabled !== false;
  const verbose = pluginConfig.verbose === true;

  if (!enabled) return;

  if (!checkTokman()) {
    console.warn("[tokman] tokman binary not found in PATH — plugin disabled");
    return;
  }

  api.on(
    "before_tool_call",
    (event: { toolName: string; params: Record<string, unknown> }) => {
      if (event.toolName !== "exec") return;

      const command = event.params?.command;
      if (typeof command !== "string") return;

      const rewritten = tryRewrite(command);
      if (!rewritten) return;

      if (verbose) {
        console.log(`[tokman] ${command} -> ${rewritten}`);
      }

      return { params: { ...event.params, command: rewritten } };
    },
    { priority: 10 }
  );

  if (verbose) {
    console.log("[tokman] OpenClaw plugin registered");
  }
}
