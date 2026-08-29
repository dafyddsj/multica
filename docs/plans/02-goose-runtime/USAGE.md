# Caller usage

The daemon does not learn Goose flags. It already knows `ResolveBackend` and `Execute`.

```go
backend, err := agent.ResolveBackend("goose", cfg)
if err != nil {
    return err
}
session, err := backend.Execute(ctx, prompt, agent.ExecOptions{
    Cwd:             workDir,
    ResumeSessionID: priorSessionID,  // "" or a parsed YYYYMMDD_* id
    ExtraArgs:       daemonGooseArgs, // MULTICA_GOOSE_ARGS
    CustomArgs:      agentCustomArgs,
    McpConfig:       mcpJSON,         // only after the isolated-extension canary
})
// Model is passed as --model only when ExecOptions.Model is set.
// A non-empty ResumeSessionID that does not parse fails Execute.
```

A fixture test owns a fake `goose` on PATH:

```go
backend, err := agent.New("goose", agent.Config{ExecutablePath: fakeGoose})
session, err := backend.Execute(ctx, "what is 2+2?", agent.ExecOptions{})
res := <-session.Result
// res.SessionID matches the parsed id the fake printed, or is empty
```

A custom runtime profile uses the family name, not a new identity:

```
protocol_family = goose
command_name    = goose
```

`InjectRuntimeConfig(workDir, "goose", ctx)` writes the brief into `AGENTS.md`. Skills land in `{workDir}/.agents/skills/`. `providerNeedsInlineSystemPrompt("goose")` stays false until a canary says otherwise.

`Result.SessionID` is only a parsed Goose id or empty. Do not store a raw unparsed token and hand it back on the next claim.
