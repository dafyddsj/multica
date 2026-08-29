# Caller usage

The daemon does not learn Amp flags. It already knows `ResolveBackend` and `Execute`.

```go
backend, err := agent.ResolveBackend("amp", cfg)
if err != nil {
    return err
}
session, err := backend.Execute(ctx, prompt, agent.ExecOptions{
    Cwd:             workDir,
    ResumeSessionID: priorSessionID,  // "" or a parsed "T-<uuid>"
    ExtraArgs:       daemonAmpArgs,   // MULTICA_AMP_ARGS
    CustomArgs:      agentCustomArgs,
    Model:           "high",          // --mode; empty uses Amp's default (medium)
    McpConfig:       mcpJSON,         // 0600 --mcp-config file plus isolate settings
})
// ThinkingLevel is ignored. A non-empty ResumeSessionID that is not T-<uuid>
// fails Execute.
```

A fixture test owns a fake `amp` on PATH:

```go
backend, err := agent.New("amp", agent.Config{ExecutablePath: fakeAmp})
session, err := backend.Execute(ctx, "what is 2+2?", agent.ExecOptions{})
res := <-session.Result
// res.SessionID == "T-f9941a55-3765-421e-972f-05dc1138c3a3"
```

A custom runtime profile uses the family name, not a new identity:

```
protocol_family = amp
command_name    = amp
```

`InjectRuntimeConfig(workDir, "amp", ctx)` writes the brief into `AGENTS.md`. Skills land in `{workDir}/.agents/skills/`. `providerNeedsInlineSystemPrompt("amp")` stays false until a canary says otherwise.

`Result.SessionID` is only a parsed `T-<uuid>` or empty. Do not store a raw unparsed `session_id` and hand it back on the next claim.
