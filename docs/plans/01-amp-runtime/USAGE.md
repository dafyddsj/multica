# Caller usage

The daemon does not learn Amp flags. It already knows `ResolveBackend` and `Execute`.

```go
backend, err := agent.ResolveBackend("amp", cfg)
if err != nil {
    return err
}
session, err := backend.Execute(ctx, prompt, agent.ExecOptions{
    Cwd:             workDir,
    Model:           model,           // optional; empty keeps Amp's default
    ThinkingLevel:   effort,          // maps to Amp effort when set
    ResumeSessionID: priorSessionID,  // "T-…" from the last Result
    ExtraArgs:       daemonAmpArgs,   // MULTICA_AMP_ARGS
    CustomArgs:      agentCustomArgs,
    McpConfig:       mcpJSON,
})
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

`InjectRuntimeConfig(workDir, "amp", ctx)` writes the brief into `AGENTS.md`. `providerNeedsInlineSystemPrompt("amp")` is false until a canary says otherwise.
