package agent

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

const ampFixtureThread = "T-f9941a55-3765-421e-972f-05dc1138c3a3"

func TestNewReturnsAmpBackend(t *testing.T) {
	t.Parallel()
	backend, err := New("amp", Config{ExecutablePath: "/nonexistent/amp"})
	if err != nil {
		t.Fatalf("New(amp): %v", err)
	}
	if _, ok := backend.(*ampBackend); !ok {
		t.Fatalf("New(amp) = %T, want *ampBackend", backend)
	}
}

func TestParseAmpThreadID(t *testing.T) {
	t.Parallel()
	if _, ok := parseAmpThreadID(ampFixtureThread); !ok {
		t.Fatalf("parseAmpThreadID(%q) = false, want true", ampFixtureThread)
	}
	if _, ok := parseAmpThreadID("T-" + strings.ToUpper(ampFixtureThread[2:])); !ok {
		t.Fatal("parseAmpThreadID should accept uppercase hex")
	}
	for _, bad := range []string{"", "sess-qwen-1", "T-not-a-uuid", ampFixtureThread[2:], "threads"} {
		if id, ok := parseAmpThreadID(bad); ok {
			t.Fatalf("parseAmpThreadID(%q) = %q, want false", bad, id)
		}
	}
}

func TestResolveAmpResume(t *testing.T) {
	t.Parallel()
	id, err := resolveAmpResume("")
	if err != nil || id != "" {
		t.Fatalf("empty resume = (%q, %v), want zero", id, err)
	}
	id, err = resolveAmpResume(ampFixtureThread)
	if err != nil || id != ampThreadID(ampFixtureThread) {
		t.Fatalf("valid resume = (%q, %v)", id, err)
	}
	if _, err := resolveAmpResume("claude-session-1"); err == nil {
		t.Fatal("malformed resume must fail")
	}
}

func TestBuildAmpArgsFreshAndResume(t *testing.T) {
	t.Parallel()
	fresh := buildAmpArgs("", ExecOptions{}, slog.Default())
	joined := strings.Join(fresh, " ")
	if strings.Contains(joined, "threads") || strings.Contains(joined, "--resume") {
		t.Fatalf("fresh argv must not resume: %v", fresh)
	}
	wantPrefix := []string{"--execute", "--stream-json", "--stream-json-thinking", "--dangerously-allow-all"}
	if len(fresh) < len(wantPrefix) {
		t.Fatalf("fresh args too short: %v", fresh)
	}
	for i, want := range wantPrefix {
		if fresh[i] != want {
			t.Fatalf("fresh[%d] = %q, want %q; all=%v", i, fresh[i], want, fresh)
		}
	}

	resume := buildAmpArgs(ampThreadID(ampFixtureThread), ExecOptions{}, slog.Default())
	wantResume := []string{"threads", "continue", ampFixtureThread, "--execute"}
	for i, want := range wantResume {
		if resume[i] != want {
			t.Fatalf("resume[%d] = %q, want %q; all=%v", i, resume[i], want, resume)
		}
	}
	if strings.Contains(strings.Join(resume, " "), "--resume") {
		t.Fatalf("resume argv must not use --resume: %v", resume)
	}
}

func TestBuildAmpArgsKeepsProtocolManaged(t *testing.T) {
	t.Parallel()
	args := buildAmpArgs("", ExecOptions{
		ExtraArgs: []string{"--execute", "--sandbox"},
		CustomArgs: []string{
			"-x", "--stream-json", "--stream-json-input", "--stream-json-thinking",
			"--dangerously-allow-all", "--mcp-config", "injected.json", "--resume", "other",
			"--continue", "--no-tui", "--executor", "orb", "-p", "prompt-text",
			"--output-format", "text", "threads", "continue", ampFixtureThread, "--debug",
		},
	}, slog.Default())
	joined := strings.Join(args, " ")
	for _, forbidden := range []string{
		"injected.json", "other", "orb", "prompt-text", "text", ampFixtureThread, "--resume", "--no-tui",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("managed argument %q leaked into %v", forbidden, args)
		}
	}
	if strings.Contains(joined, "threads") {
		t.Fatalf("threads continue sequence leaked into %v", args)
	}
	if count := strings.Count(joined, "--execute"); count != 1 {
		t.Fatalf("--execute count = %d in %v, want 1", count, args)
	}
	if count := strings.Count(joined, "--dangerously-allow-all"); count != 1 {
		t.Fatalf("--dangerously-allow-all count = %d in %v, want 1", count, args)
	}
	if !strings.Contains(joined, "--sandbox") || !strings.Contains(joined, "--debug") {
		t.Fatalf("non-managed custom args missing from %v", args)
	}
}

func TestFilterAmpThreadSequencesDropsWholeRun(t *testing.T) {
	t.Parallel()
	got := filterAmpThreadSequences([]string{"--foo", "threads", "continue", ampFixtureThread, "--bar"}, slog.Default())
	want := []string{"--foo", "--bar"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	got = filterAmpThreadSequences([]string{"threads", "continue", "--execute"}, slog.Default())
	if strings.Contains(strings.Join(got, " "), "threads") {
		t.Fatalf("orphan threads continue left tokens: %v", got)
	}
}

func fakeAmpScript() string {
	return `#!/bin/sh
if [ -n "$AMP_ARGS_FILE" ]; then printf '%s\n' "$@" > "$AMP_ARGS_FILE"; fi
if [ -n "$AMP_STDIN_FILE" ]; then cat > "$AMP_STDIN_FILE"; fi
case "$AMP_MODE" in
  error)
    printf '%s\n' '{"type":"system","subtype":"init","session_id":"T-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}'
    printf '%s\n' '{"type":"result","subtype":"error_during_execution","session_id":"T-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","is_error":true,"error":{"message":"synthetic amp failure"}}'
    ;;
  resume-mismatch)
    printf '%s\n' '{"type":"system","subtype":"init","session_id":"T-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"}'
    printf '%s\n' '{"type":"result","subtype":"error_during_execution","session_id":"T-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","is_error":true,"result":"thread not found"}'
    exit 1
    ;;
  garbage-session)
    printf '%s\n' '{"type":"system","subtype":"init","session_id":"not-a-thread"}'
    printf '%s\n' '{"type":"assistant","session_id":"not-a-thread","message":{"role":"assistant","content":[{"type":"text","text":"ok"}]}}'
    printf '%s\n' '{"type":"result","subtype":"success","session_id":"not-a-thread","is_error":false,"result":"ok"}'
    ;;
  exit)
    echo 'synthetic amp stderr' >&2
    exit 7
    ;;
  *)
    printf '%s\n' '{"type":"system","subtype":"init","session_id":"T-f9941a55-3765-421e-972f-05dc1138c3a3","cwd":"/work","tools":[]}'
    printf '%s\n' '{"type":"assistant","session_id":"T-f9941a55-3765-421e-972f-05dc1138c3a3","message":{"role":"assistant","model":"amp-test","content":[{"type":"thinking","thinking":"planning"}]}}'
    printf '%s\n' '{"type":"assistant","session_id":"T-f9941a55-3765-421e-972f-05dc1138c3a3","message":{"role":"assistant","model":"amp-test","content":[{"type":"tool_use","id":"call-1","name":"list_directory","input":{"path":"/work"}}]}}'
    printf '%s\n' '{"type":"user","session_id":"T-f9941a55-3765-421e-972f-05dc1138c3a3","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"call-1","content":"Listed 1 item"}]}}'
    printf '%s\n' '{"type":"assistant","session_id":"T-f9941a55-3765-421e-972f-05dc1138c3a3","message":{"role":"assistant","model":"amp-test","content":[{"type":"redacted_thinking"},{"type":"text","text":"PONG"}],"usage":{"input_tokens":10,"output_tokens":2,"cache_read_input_tokens":3}}}'
    printf '%s\n' '{"type":"result","subtype":"success","session_id":"T-f9941a55-3765-421e-972f-05dc1138c3a3","is_error":false,"result":"PONG","usage":{"input_tokens":20,"output_tokens":4,"cache_read_input_tokens":6}}'
    ;;
esac
`
}

func newFakeAmpBackend(t *testing.T, env map[string]string) *ampBackend {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	path := filepath.Join(t.TempDir(), "amp")
	writeTestExecutable(t, path, []byte(fakeAmpScript()))
	return &ampBackend{cfg: Config{ExecutablePath: path, Logger: slog.Default(), Env: env}}
}

func awaitAmpResult(t *testing.T, session *Session) ([]Message, Result) {
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
		t.Fatal("timed out waiting for amp result")
		return nil, Result{}
	}
}

func TestAmpBackendStreamsNativeEvents(t *testing.T) {
	t.Parallel()
	backend := newFakeAmpBackend(t, nil)
	session, err := backend.Execute(context.Background(), "reply PONG", ExecOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	messages, result := awaitAmpResult(t, session)
	if result.Status != "completed" || result.Output != "PONG" || result.SessionID != ampFixtureThread {
		t.Fatalf("unexpected result: %+v", result)
	}
	usage := result.Usage["amp-test"]
	if usage.InputTokens != 20 || usage.OutputTokens != 4 || usage.CacheReadTokens != 6 {
		t.Fatalf("unexpected final usage: %+v", usage)
	}
	var thinking, toolUse, toolResult, text bool
	for _, message := range messages {
		switch message.Type {
		case MessageThinking:
			thinking = message.Content == "planning"
		case MessageToolUse:
			toolUse = message.Tool == "list_directory" && message.CallID == "call-1" && message.Input["path"] == "/work"
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

func TestAmpBackendResumeUsesThreadsContinue(t *testing.T) {
	t.Parallel()
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	backend := newFakeAmpBackend(t, map[string]string{"AMP_ARGS_FILE": argsFile})
	session, err := backend.Execute(context.Background(), "continue please", ExecOptions{
		ResumeSessionID: ampFixtureThread,
		Timeout:         5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	_, result := awaitAmpResult(t, session)
	if result.SessionID != ampFixtureThread {
		t.Fatalf("session id = %q", result.SessionID)
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	argv := strings.Fields(string(got))
	wantPrefix := []string{"threads", "continue", ampFixtureThread, "--execute"}
	if len(argv) < len(wantPrefix) {
		t.Fatalf("argv too short: %v", argv)
	}
	for i, want := range wantPrefix {
		if argv[i] != want {
			t.Fatalf("argv[%d] = %q, want %q; all=%v", i, argv[i], want, argv)
		}
	}
	for _, a := range argv {
		if a == "--resume" {
			t.Fatalf("--resume leaked into argv: %v", argv)
		}
	}
}

func TestAmpBackendRejectsMalformedResume(t *testing.T) {
	t.Parallel()
	backend := newFakeAmpBackend(t, nil)
	_, err := backend.Execute(context.Background(), "go", ExecOptions{ResumeSessionID: "claude-uuid"})
	if err == nil {
		t.Fatal("malformed ResumeSessionID must fail Execute")
	}
}

func TestAmpBackendDropsUnparseableSessionID(t *testing.T) {
	t.Parallel()
	backend := newFakeAmpBackend(t, map[string]string{"AMP_MODE": "garbage-session"})
	session, err := backend.Execute(context.Background(), "go", ExecOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	_, result := awaitAmpResult(t, session)
	if result.SessionID != "" {
		t.Fatalf("unparseable session_id persisted as %q", result.SessionID)
	}
}

func TestAmpBackendResumeMismatchSetsRejected(t *testing.T) {
	t.Parallel()
	backend := newFakeAmpBackend(t, map[string]string{"AMP_MODE": "resume-mismatch"})
	session, err := backend.Execute(context.Background(), "continue", ExecOptions{
		ResumeSessionID: ampFixtureThread,
		Timeout:         5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	_, result := awaitAmpResult(t, session)
	if result.Status != "failed" || !result.ResumeRejected {
		t.Fatalf("want failed+ResumeRejected, got %+v", result)
	}
	if result.SessionID != "" {
		t.Fatalf("rejected resume persisted %q", result.SessionID)
	}
}

func TestAmpBackendDeliversPromptOnStdin(t *testing.T) {
	t.Parallel()
	stdinPath := filepath.Join(t.TempDir(), "amp.stdin")
	backend := newFakeAmpBackend(t, map[string]string{"AMP_STDIN_FILE": stdinPath})
	prompt := `go build -ldflags "-X main.version=foo" — mind the quotes"`
	session, err := backend.Execute(context.Background(), prompt, ExecOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	_, result := awaitAmpResult(t, session)
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

func TestAmpExtraArgsReachTheCommandLine(t *testing.T) {
	t.Parallel()
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	backend := newFakeAmpBackend(t, map[string]string{"AMP_ARGS_FILE": argsFile})
	session, err := backend.Execute(context.Background(), "test prompt", ExecOptions{
		Timeout:    5 * time.Second,
		ExtraArgs:  []string{"--daemon-wide"},
		CustomArgs: []string{"--per-agent"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	_, result := awaitAmpResult(t, session)
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

func TestAmpFixtureParses(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("testdata", "amp-stream-json.jsonl"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	state := ampStreamState{usage: make(map[string]TokenUsage)}
	ch := make(chan Message, 32)
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var event ampStreamEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("unmarshal %q: %v", line, err)
		}
		handleAmpEvent(event, ch, &state)
	}
	close(ch)
	if state.threadID != ampThreadID(ampFixtureThread) {
		t.Fatalf("thread id = %q", state.threadID)
	}
	if state.resultIsError || !state.sawResult || state.finalResultText != "PONG" {
		t.Fatalf("fixture terminal state: %+v", state)
	}
}

func TestAmpListModelsIsEmptyWithoutSpawning(t *testing.T) {
	t.Parallel()
	cat, err := ListModels(context.Background(), "amp", Command{Path: "/nonexistent/amp"})
	if err != nil {
		t.Fatalf("ListModels(amp): %v", err)
	}
	if len(cat.Models) != 0 {
		t.Fatalf("ListModels(amp) = %d models, want empty", len(cat.Models))
	}
	if ModelSelectionSupported("amp") {
		t.Fatal("ModelSelectionSupported(amp) should be false")
	}
}

func TestChooseAmpInvocation_PassthroughForNonLauncher(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	execName := "amp"
	lookedUp := filepath.Join(t.TempDir(), "amp")
	args := []string{"--execute", "--stream-json"}
	gotExec, gotArgs := chooseAmpInvocation(execName, lookedUp, args, logger)
	if gotExec != execName {
		t.Errorf("argv0 changed: got %q want %q", gotExec, execName)
	}
	if !reflect.DeepEqual(gotArgs, args) {
		t.Errorf("argv changed:\n got  %#v\n want %#v", gotArgs, args)
	}
}

func TestAmpLaunchesThroughTheInvocationChooser(t *testing.T) {
	t.Parallel()
	file, err := parser.ParseFile(token.NewFileSet(), "amp.go", nil, 0)
	if err != nil {
		t.Fatalf("parse amp.go: %v", err)
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
			if ident, ok := arg.(*ast.Ident); ok && ident.Name == "chooseAmpInvocation" {
				routed = true
			}
		}
		return true
	})
	if !routed {
		t.Fatal("amp.go must spawn through Command.execVia with chooseAmpInvocation")
	}
}
