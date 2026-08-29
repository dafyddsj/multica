package agent

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

const gooseFixtureSession = "20250325_200615"
const gooseAltSession = "20251108_1"

func TestNewReturnsGooseBackend(t *testing.T) {
	t.Parallel()
	backend, err := New("goose", Config{ExecutablePath: "/nonexistent/goose"})
	if err != nil {
		t.Fatalf("New(goose): %v", err)
	}
	if _, ok := backend.(*gooseBackend); !ok {
		t.Fatalf("New(goose) = %T, want *gooseBackend", backend)
	}
}

func TestParseGooseSessionID(t *testing.T) {
	t.Parallel()
	if _, ok := parseGooseSessionID(gooseFixtureSession); !ok {
		t.Fatalf("parseGooseSessionID(%q) = false, want true", gooseFixtureSession)
	}
	if _, ok := parseGooseSessionID(gooseAltSession); !ok {
		t.Fatalf("parseGooseSessionID(%q) = false, want true", gooseAltSession)
	}
	for _, bad := range []string{"", "sess-qwen-1", "T-not-a-uuid", "20250325", "20250325-", "session_1", "2025_1"} {
		if id, ok := parseGooseSessionID(bad); ok {
			t.Fatalf("parseGooseSessionID(%q) = %q, want false", bad, id)
		}
	}
}

func TestResolveGooseResume(t *testing.T) {
	t.Parallel()
	id, err := resolveGooseResume("")
	if err != nil || id != "" {
		t.Fatalf("empty resume = (%q, %v), want zero", id, err)
	}
	id, err = resolveGooseResume(gooseFixtureSession)
	if err != nil || id != gooseSessionID(gooseFixtureSession) {
		t.Fatalf("valid resume = (%q, %v)", id, err)
	}
	if _, err := resolveGooseResume("claude-session-1"); err == nil {
		t.Fatal("malformed resume must fail")
	}
}

func TestBuildGooseArgsFreshAndResume(t *testing.T) {
	t.Parallel()
	fresh := buildGooseArgs("", ExecOptions{}, slog.Default())
	wantFresh := []string{"run", "-i", "-", "--output-format", "stream-json"}
	if !reflect.DeepEqual(fresh, wantFresh) {
		t.Fatalf("fresh = %v, want %v", fresh, wantFresh)
	}
	joined := strings.Join(fresh, " ")
	if strings.Contains(joined, "--resume") || strings.Contains(joined, "--session-id") {
		t.Fatalf("fresh argv must not resume: %v", fresh)
	}
	if strings.Contains(joined, "-t") || strings.Contains(joined, "--text") {
		t.Fatalf("fresh argv must not put the prompt on -t: %v", fresh)
	}

	withModel := buildGooseArgs("", ExecOptions{Model: "gpt-4o"}, slog.Default())
	if !strings.Contains(strings.Join(withModel, " "), "--model gpt-4o") {
		t.Fatalf("fresh with model missing --model gpt-4o: %v", withModel)
	}

	withProvider := buildGooseArgs("", ExecOptions{GooseProvider: "ollama", Model: "llama3.2"}, slog.Default())
	if !strings.Contains(strings.Join(withProvider, " "), "--provider ollama --model llama3.2") {
		t.Fatalf("fresh with provider missing --provider ollama --model llama3.2: %v", withProvider)
	}

	emptyProvider := buildGooseArgs("", ExecOptions{GooseProvider: "  ", Model: "llama3.2"}, slog.Default())
	if strings.Contains(strings.Join(emptyProvider, " "), "--provider") {
		t.Fatalf("empty GooseProvider must omit --provider: %v", emptyProvider)
	}

	resume := buildGooseArgs(gooseSessionID(gooseFixtureSession), ExecOptions{}, slog.Default())
	wantResume := []string{"run", "-i", "-", "--output-format", "stream-json", "--resume", "--session-id", gooseFixtureSession}
	if !reflect.DeepEqual(resume, wantResume) {
		t.Fatalf("resume = %v, want %v", resume, wantResume)
	}
	resumeJoined := strings.Join(resume, " ")
	if strings.Count(resumeJoined, "--resume") != 1 {
		t.Fatalf("resume flag count wrong: %v", resume)
	}
	if !strings.Contains(resumeJoined, "--session-id "+gooseFixtureSession) {
		t.Fatalf("resume missing --session-id: %v", resume)
	}
}

func TestBuildGooseArgsKeepsProtocolManaged(t *testing.T) {
	t.Parallel()
	args := buildGooseArgs("", ExecOptions{
		ExtraArgs: []string{"--debug", "-i", "evil.md"},
		CustomArgs: []string{
			"-t", "prompt-text", "--text", "more-text",
			"--output-format", "text", "--resume", "--session-id", gooseFixtureSession,
			"--no-session", "-s", "--interactive", "--recipe", "demo.yaml",
			"--acp", "acp", "serve", "--instructions", "file.md", "--max-turns", "8",
			"--provider", "openai",
		},
	}, slog.Default())
	joined := strings.Join(args, " ")
	for _, forbidden := range []string{
		"evil.md", "prompt-text", "more-text", "text", gooseFixtureSession, "demo.yaml", "file.md",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("managed argument %q leaked into %v", forbidden, args)
		}
	}
	if strings.Contains(joined, "--resume") || strings.Contains(joined, "--session-id") {
		t.Fatalf("resume tokens leaked into fresh argv: %v", args)
	}
	if strings.Contains(joined, "--no-session") || strings.Contains(joined, "--interactive") {
		t.Fatalf("session-discard tokens leaked into %v", args)
	}
	if count := strings.Count(joined, "--output-format"); count != 1 {
		t.Fatalf("--output-format count = %d in %v, want 1", count, args)
	}
	if !strings.Contains(joined, "--debug") || !strings.Contains(joined, "--max-turns") {
		t.Fatalf("non-managed custom args missing from %v", args)
	}
	if strings.Contains(joined, "--provider") || strings.Contains(joined, "openai") {
		t.Fatalf("custom --provider leaked into %v", args)
	}
}

func TestBuildGooseArgsBlocksProviderInExtraArgs(t *testing.T) {
	t.Parallel()
	args := buildGooseArgs("", ExecOptions{
		GooseProvider: "ollama",
		ExtraArgs:     []string{"--provider", "openai", "--debug"},
	}, slog.Default())
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "openai") {
		t.Fatalf("ExtraArgs --provider leaked into %v", args)
	}
	if !strings.Contains(joined, "--provider ollama") {
		t.Fatalf("first-class GooseProvider missing from %v", args)
	}
	if !strings.Contains(joined, "--debug") {
		t.Fatalf("non-managed ExtraArgs missing from %v", args)
	}
}

func TestFilterGooseBlockedArgs(t *testing.T) {
	t.Parallel()
	got := filterGooseRuntimeArgs([]string{"--debug", "serve", "--foo", "acp", "--bar"}, slog.Default())
	want := []string{"--debug", "--foo", "--bar"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func fakeGooseScript() string {
	return `#!/bin/sh
if [ -n "$GOOSE_ARGS_FILE" ]; then printf '%s\n' "$@" > "$GOOSE_ARGS_FILE"; fi
if [ -n "$GOOSE_ENV_FILE" ]; then printf '%s\n' "$GOOSE_MODE" > "$GOOSE_ENV_FILE"; fi
if [ -n "$GOOSE_STDIN_FILE" ]; then cat > "$GOOSE_STDIN_FILE"; else cat >/dev/null; fi
case "$GOOSE_MODE_FAKE" in
  error)
    printf '%s\n' '{"type":"error","error":"synthetic goose failure"}'
    ;;
  resume-mismatch)
    printf '%s\n' '{"type":"error","error":"No conversation found","session_id":"20251108_1"}'
    exit 1
    ;;
  garbage-session)
    printf '%s\n' '{"type":"message","session_id":"not-a-session","message":{"role":"assistant","created":1,"content":[{"type":"text","text":"ok"}],"metadata":{"userVisible":true,"agentVisible":true}}}'
    printf '%s\n' '{"type":"complete","total_tokens":1}'
    ;;
  exit)
    echo 'synthetic goose stderr' >&2
    exit 7
    ;;
  *)
    printf '%s\n' '{"type":"notification","extension_id":"developer","message":"running"}'
    printf '%s\n' '{"type":"message","message":{"role":"assistant","created":1,"content":[{"type":"thinking","thinking":"planning","signature":""}],"metadata":{"userVisible":true,"agentVisible":true}}}'
    printf '%s\n' '{"type":"message","message":{"role":"assistant","created":2,"content":[{"type":"toolRequest","id":"call-1","toolCall":{"name":"developer__shell","arguments":{"command":"ls"}}}],"metadata":{"userVisible":true,"agentVisible":true,"inference":{"provider":"openai","requestedModel":"goose-test","resolvedModel":"goose-test"}}}}'
    printf '%s\n' '{"type":"message","message":{"role":"user","created":3,"content":[{"type":"toolResponse","id":"call-1","toolResult":"Listed 1 item"}],"metadata":{"userVisible":true,"agentVisible":true}}}'
    printf '%s\n' '{"type":"message","message":{"role":"assistant","created":4,"content":[{"type":"redactedThinking","data":""},{"type":"text","text":"PONG"}],"metadata":{"userVisible":true,"agentVisible":true,"inference":{"provider":"openai","requestedModel":"goose-test","resolvedModel":"goose-test"},"usage":{"inputTokens":10,"outputTokens":2,"cacheReadTokens":3}}}}'
    printf '%s\n' '{"type":"complete","total_tokens":24,"input_tokens":20,"output_tokens":4,"cache_read_input_tokens":6}'
    ;;
esac
`
}

func newFakeGooseBackend(t *testing.T, env map[string]string) *gooseBackend {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	path := filepath.Join(t.TempDir(), "goose")
	writeTestExecutable(t, path, []byte(fakeGooseScript()))
	return &gooseBackend{cfg: Config{ExecutablePath: path, Logger: slog.Default(), Env: env}}
}

func awaitGooseResult(t *testing.T, session *Session) ([]Message, Result) {
	t.Helper()
	var messages []Message
	for message := range session.Messages {
		messages = append(messages, message)
	}
	select {
	case result, ok := <-session.Result:
		if !ok {
			t.Fatal("result channel closed without a result")
		}
		return messages, result
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for goose result")
		return nil, Result{}
	}
}

func TestGooseBackendStreamsNativeEvents(t *testing.T) {
	t.Parallel()
	backend := newFakeGooseBackend(t, nil)
	session, err := backend.Execute(context.Background(), "reply PONG", ExecOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	messages, result := awaitGooseResult(t, session)
	if result.Status != "completed" || result.Output != "PONG" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.SessionID != "" {
		t.Fatalf("fresh run persisted unemitted session id %q", result.SessionID)
	}
	usage := result.Usage["goose-test"]
	if usage.InputTokens != 20 || usage.OutputTokens != 4 || usage.CacheReadTokens != 6 {
		t.Fatalf("unexpected final usage: %+v", usage)
	}
	var thinking, toolUse, toolResult, text bool
	for _, message := range messages {
		switch message.Type {
		case MessageThinking:
			thinking = message.Content == "planning"
		case MessageToolUse:
			toolUse = message.Tool == "developer__shell" && message.CallID == "call-1" && message.Input["command"] == "ls"
		case MessageToolResult:
			toolResult = message.CallID == "call-1" && message.Output == "Listed 1 item"
		case MessageText:
			text = message.Content == "PONG"
		}
	}
	if !thinking || !toolUse || !toolResult || !text {
		t.Fatalf("missing native events thinking=%v toolUse=%v toolResult=%v text=%v; messages=%+v", thinking, toolUse, toolResult, text, messages)
	}
}

func TestGooseBackendResumeUsesResumeAndSessionID(t *testing.T) {
	t.Parallel()
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	backend := newFakeGooseBackend(t, map[string]string{"GOOSE_ARGS_FILE": argsFile})
	session, err := backend.Execute(context.Background(), "continue please", ExecOptions{
		ResumeSessionID: gooseFixtureSession,
		Timeout:         5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	_, result := awaitGooseResult(t, session)
	if result.SessionID != gooseFixtureSession {
		t.Fatalf("session id = %q, want the parsed resume id", result.SessionID)
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	argv := strings.Fields(string(got))
	wantPrefix := []string{"run", "-i", "-", "--output-format", "stream-json", "--resume", "--session-id", gooseFixtureSession}
	if len(argv) < len(wantPrefix) {
		t.Fatalf("argv too short: %v", argv)
	}
	for i, want := range wantPrefix {
		if argv[i] != want {
			t.Fatalf("argv[%d] = %q, want %q; all=%v", i, argv[i], want, argv)
		}
	}
}

func TestGooseBackendRejectsMalformedResume(t *testing.T) {
	t.Parallel()
	backend := newFakeGooseBackend(t, nil)
	_, err := backend.Execute(context.Background(), "go", ExecOptions{ResumeSessionID: "claude-uuid"})
	if err == nil {
		t.Fatal("malformed ResumeSessionID must fail Execute")
	}
}

func TestGooseBackendDropsUnparseableSessionID(t *testing.T) {
	t.Parallel()
	backend := newFakeGooseBackend(t, map[string]string{"GOOSE_MODE_FAKE": "garbage-session"})
	session, err := backend.Execute(context.Background(), "go", ExecOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	_, result := awaitGooseResult(t, session)
	if result.SessionID != "" {
		t.Fatalf("unparseable session_id persisted as %q", result.SessionID)
	}
}

func TestGooseBackendResumeMismatchSetsRejected(t *testing.T) {
	t.Parallel()
	backend := newFakeGooseBackend(t, map[string]string{"GOOSE_MODE_FAKE": "resume-mismatch"})
	session, err := backend.Execute(context.Background(), "continue", ExecOptions{
		ResumeSessionID: gooseFixtureSession,
		Timeout:         5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	_, result := awaitGooseResult(t, session)
	if result.Status != "failed" || !result.ResumeRejected {
		t.Fatalf("want failed+ResumeRejected, got %+v", result)
	}
	if result.SessionID != "" {
		t.Fatalf("rejected resume persisted %q", result.SessionID)
	}
}

func TestGooseBackendDeliversPromptOnStdin(t *testing.T) {
	t.Parallel()
	stdinPath := filepath.Join(t.TempDir(), "goose.stdin")
	backend := newFakeGooseBackend(t, map[string]string{"GOOSE_STDIN_FILE": stdinPath})
	prompt := `go build -ldflags "-X main.version=foo" — mind the quotes"`
	session, err := backend.Execute(context.Background(), prompt, ExecOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	_, result := awaitGooseResult(t, session)
	if result.Status != "completed" {
		t.Fatalf("result = %+v", result)
	}
	got, err := os.ReadFile(stdinPath)
	if err != nil {
		t.Fatalf("read captured stdin: %v", err)
	}
	if string(got) != prompt {
		t.Fatalf("stdin content = %q, want %q", got, prompt)
	}
}

func TestGooseBackendPinsGooseModeAuto(t *testing.T) {
	t.Parallel()
	envPath := filepath.Join(t.TempDir(), "goose.env")
	backend := newFakeGooseBackend(t, map[string]string{"GOOSE_ENV_FILE": envPath})
	session, err := backend.Execute(context.Background(), "go", ExecOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	_, result := awaitGooseResult(t, session)
	if result.Status != "completed" {
		t.Fatalf("result = %+v", result)
	}
	got, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read env: %v", err)
	}
	if strings.TrimSpace(string(got)) != "auto" {
		t.Fatalf("GOOSE_MODE = %q, want auto", got)
	}
}

func TestGooseExtraArgsReachTheCommandLine(t *testing.T) {
	t.Parallel()
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	backend := newFakeGooseBackend(t, map[string]string{"GOOSE_ARGS_FILE": argsFile})
	session, err := backend.Execute(context.Background(), "test prompt", ExecOptions{
		Timeout:    5 * time.Second,
		ExtraArgs:  []string{"--daemon-wide"},
		CustomArgs: []string{"--per-agent"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	_, result := awaitGooseResult(t, session)
	if result.Status != "completed" {
		t.Fatalf("result = %+v", result)
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	joined := string(got)
	daemonIdx := strings.Index(joined, "--daemon-wide")
	agentIdx := strings.Index(joined, "--per-agent")
	if daemonIdx < 0 || agentIdx < 0 || daemonIdx > agentIdx {
		t.Fatalf("ExtraArgs must precede CustomArgs in %q", joined)
	}
}

func TestGooseFixtureParses(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("testdata", "goose-stream-json.jsonl"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	state := gooseStreamState{usage: make(map[string]TokenUsage)}
	ch := make(chan Message, 32)
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var event gooseStreamEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("unmarshal %q: %v", line, err)
		}
		handleGooseEvent(event, ch, &state)
	}
	close(ch)
	if state.sessionID != "" {
		t.Fatalf("fixture invented a session id %q", state.sessionID)
	}
	if state.resultIsError || !state.sawResult || state.lastAssistantText != "PONG" {
		t.Fatalf("fixture terminal state: %+v", state)
	}
}

func TestGooseListModelsReturnsEmptyWithoutSpawning(t *testing.T) {
	t.Parallel()
	cat, err := ListModels(context.Background(), "goose", Command{Path: "/nonexistent/goose"})
	if err != nil {
		t.Fatalf("ListModels(goose): %v", err)
	}
	if len(cat.Models) != 0 {
		t.Fatalf("ListModels(goose) = %#v, want empty catalog", cat.Models)
	}
	if !ModelSelectionSupported("goose") {
		t.Fatal("ModelSelectionSupported(goose) should be true: goose honors --model like Qwen (empty catalog, manual entry)")
	}
}

func TestGooseLaunchesThroughTheInvocationChooser(t *testing.T) {
	t.Parallel()
	file, err := parser.ParseFile(token.NewFileSet(), "goose.go", nil, 0)
	if err != nil {
		t.Fatalf("parse goose.go: %v", err)
	}
	routed := false
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "execVia" {
			return true
		}
		for _, arg := range call.Args {
			if ident, ok := arg.(*ast.Ident); ok && ident.Name == "chooseGooseInvocation" {
				routed = true
			}
		}
		return true
	})
	if !routed {
		t.Fatal("goose.go must spawn through Command.execVia with chooseGooseInvocation")
	}
}
