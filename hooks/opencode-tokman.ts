import type { Plugin } from "@opencode-ai/plugin"

// TokMan OpenCode plugin — rewrites commands to use tokman for token savings.

export const TokmanOpenCodePlugin: Plugin = async ({ $ }) => {
  try {
    await $`which tokman`.quiet()
  } catch {
    console.warn("[tokman] tokman binary not found in PATH — plugin disabled")
    return {}
  }

  return {
    "tool.execute.before": async (input, output) => {
      const tool = String(input?.tool ?? "").toLowerCase()
      if (tool !== "bash" && tool !== "shell") return
      const args = output?.args
      if (!args || typeof args !== "object") return

      const command = (args as Record<string, unknown>).command
      if (typeof command !== "string" || !command) return

      try {
        const result = await $`tokman rewrite ${command}`.quiet().nothrow()
        const rewritten = String(result.stdout).trim()
        if (rewritten && rewritten !== command) {
          ;(args as Record<string, unknown>).command = rewritten
        }
      } catch {
        // tokman rewrite failed — pass through unchanged
      }
    },
  }
}
