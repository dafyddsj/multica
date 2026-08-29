# Caller usage

The daemon does not learn Devin flags. It already knows `ResolveBackend` and `Execute`.

```go
backend, err := agent.ResolveBackend("devin", cfg)
if err != nil {
    return err
}
session, err := backend.Execute(ctx, prompt, agent.ExecOptions{
    Cwd:             workDir,
    ResumeSessionID: priorSessionID,  // "" or a parsed CLI session id
    ExtraArgs:       daemonDevinArgs, // MULTICA_DEVIN_ARGS
    CustomArgs:      agentCustomArgs,
    McpConfig:       mcpJSON,         // only after the isolated-config canary
})
// Model is passed as --model only when ExecOptions.Model is set.
// A non-empty ResumeSessionID that does not parse fails Execute.
```

A fixture test owns a fake `devin` on PATH:

```go
backend, err := agent.New("devin", agent.Config{ExecutablePath: fakeDevin})
session, err := backend.Execute(ctx, "what is 2+2?", agent.ExecOptions{})
res := <-session.Result
// res.SessionID matches the parsed id the fake printed, or is empty
```

A custom runtime profile uses the family name, not a new identity:

```
protocol_family = devin
command_name    = devin
```

`InjectRuntimeConfig(workDir, "devin", ctx)` writes the brief into `AGENTS.md`. Skills land in `{workDir}/.devin/skills/`. `providerNeedsInlineSystemPrompt("devin")` stays false until a canary says otherwise.

`Result.SessionID` is only a parsed Devin CLI id or empty. Do not store a raw unparsed token and hand it back on the next claim. Do not store a cloud `devin-…` box id as a CLI resume id.
