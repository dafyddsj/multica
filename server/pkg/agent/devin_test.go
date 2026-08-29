package agent

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

const devinFixtureSession = "brisk-otter"

func TestNewReturnsDevinBackend(t *testing.T) {
	t.Parallel()
	backend, err := New("devin", Config{ExecutablePath: "/nonexistent/devin"})
	if err != nil {
		t.Fatalf("New(devin): %v", err)
	}
	if _, ok := backend.(*devinBackend); !ok {
		t.Fatalf("New(devin) = %T, want *devinBackend", backend)
	}
}

func TestParseDevinSessionID(t *testing.T) {
	t.Parallel()
	for _, good := range []string{devinFixtureSession, "abc12345", "Otter_1"} {
		if _, ok := parseDevinSessionID(good); !ok {
			t.Fatalf("parseDevinSessionID(%q) = false, want true", good)
		}
	}
	for _, bad := range []string{
		"", "-", "--resume", "devin-abc123", "devin-abc12345",
		"/tmp/session", "https://devin.ai/s", "has space", "42", "4",
	} {
		if id, ok := parseDevinSessionID(bad); ok {
			t.Fatalf("parseDevinSessionID(%q) = %q, want false", bad, id)
		}
	}
}

func TestResolveDevinResume(t *testing.T) {
	t.Parallel()
	id, err := resolveDevinResume("")
	if err != nil || id != "" {
		t.Fatalf("empty resume = (%q, %v), want zero", id, err)
	}
	id, err = resolveDevinResume(devinFixtureSession)
	if err != nil || id != devinSessionID(devinFixtureSession) {
		t.Fatalf("valid resume = (%q, %v)", id, err)
	}
	if _, err := resolveDevinResume("devin-abc123"); err == nil {
		t.Fatal("cloud-shaped resume must fail")
	}
	if _, err := resolveDevinResume("--resume"); err == nil {
		t.Fatal("flag-like resume must fail")
	}
}

func TestExtractDevinSessionID(t *testing.T) {
	t.Parallel()
	if got := extractDevinSessionID("session: " + devinFixtureSession + "\n4\n"); got != devinFixtureSession {
		t.Fatalf("labeled extract = %q, want %q", got, devinFixtureSession)
	}
	if got := extractDevinSessionID("the answer is 42"); got != "" {
		t.Fatalf("numeric answer must not become a session id, got %q", got)
	}
}

func TestBuildDevinArgsFreshAndResume(t *testing.T) {
	t.Parallel()
	fresh := buildDevinArgs("", ExecOptions{}, "/tmp/prompt.txt", slog.Default())
	joined := strings.Join(fresh, " ")
	want := []string{"--print", "--prompt-file", "/tmp/prompt.txt", "--permission-mode", "dangerous", "--respect-workspace-trust", "false"}
	if len(fresh) < len(want) {
		t.Fatalf("fresh args too short: %v", fresh)
	}
	for i, token := range want {
		if fresh[i] != token {
			t.Fatalf("fresh[%d] = %q, want %q; all=%v", i, fresh[i], token, fresh)
		}
	}
	if strings.Contains(joined, "--resume") || strings.Contains(joined, "-c") || strings.Contains(joined, "--continue") {
		t.Fatalf("fresh argv must not resume: %v", fresh)
	}

	withModel := buildDevinArgs("", ExecOptions{Model: "opus"}, "/tmp/prompt.txt", slog.Default())
	if !strings.Contains(strings.Join(withModel, " "), "--model opus") {
		t.Fatalf("model missing from %v", withModel)
	}

	resume := buildDevinArgs(devinSessionID(devinFixtureSession), ExecOptions{}, "/tmp/prompt.txt", slog.Default())
	joinedResume := strings.Join(resume, " ")
	if !strings.Contains(joinedResume, "--resume "+devinFixtureSession) {
		t.Fatalf("resume argv missing --resume <id>: %v", resume)
	}
	if strings.Contains(joinedResume, "-c") || strings.Contains(joinedResume, "--continue") {
		t.Fatalf("resume argv must not use --continue: %v", resume)
	}
	if strings.Count(joinedResume, "--resume") != 1 {
		t.Fatalf("expected one --resume in %v", resume)
	}
}

func TestBuildDevinArgsKeepsProtocolManaged(t *testing.T) {
	t.Parallel()
	args := buildDevinArgs("", ExecOptions{
		ExtraArgs: []string{"--verbose"},
		CustomArgs: []string{
			"-p", "inline-prompt", "--print", "also-inline",
			"--prompt-file", "evil.txt", "--permission-mode", "auto",
			"--respect-workspace-trust", "true", "--model", "hijack",
			"-c", "--continue", "-r", "other", "--resume", "other",
			"--sandbox", "acp", "cloud", "ssh", "desktop", "--debug",
		},
	}, "/tmp/prompt.txt", slog.Default())
	joined := strings.Join(args, " ")
	for _, forbidden := range []string{
		"inline-prompt", "also-inline", "evil.txt", "auto", "true", "hijack",
		"other", "acp", "cloud", "ssh", "desktop", "-c", "--continue",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("managed argument %q leaked into %v", forbidden, args)
		}
	}
	if !strings.Contains(joined, "--verbose") || !strings.Contains(joined, "--debug") {
		t.Fatalf("non-managed custom args missing from %v", args)
	}
	if strings.Count(joined, "--print") != 1 {
		t.Fatalf("--print count = %d in %v, want 1", strings.Count(joined, "--print"), args)
	}
	if strings.Count(joined, "--permission-mode") != 1 || !strings.Contains(joined, "dangerous") {
		t.Fatalf("permission mode drifted: %v", args)
	}
}

func TestBuildDevinArgsExtraArgsPrecedeCustomArgs(t *testing.T) {
	t.Parallel()
	args := buildDevinArgs("", ExecOptions{
		ExtraArgs:  []string{"--from-extra"},
		CustomArgs: []string{"--from-custom"},
	}, "/tmp/prompt.txt", slog.Default())
	extraAt, customAt := -1, -1
	for i, arg := range args {
		if arg == "--from-extra" {
			extraAt = i
		}
		if arg == "--from-custom" {
			customAt = i
		}
	}
	if extraAt < 0 || customAt < 0 || extraAt > customAt {
		t.Fatalf("ExtraArgs must precede CustomArgs in %v", args)
	}
}

func fakeDevinScript() string {
	return `#!/bin/sh
if [ -n "$DEVIN_ARGS_FILE" ]; then printf '%s\n' "$@" > "$DEVIN_ARGS_FILE"; fi
prompt=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--prompt-file" ]; then prompt="$2"; fi
  if [ "$1" = "-p" ] || [ "$1" = "--print" ]; then
    case "$2" in
      --*|"") ;;
      *) printf '%s\n' "$2" > "${DEVIN_INLINE_PROMPT_FILE:-/tmp/devin-inline-unused}" ;;
    esac
  fi
  shift
done
if [ -n "$DEVIN_PROMPT_CAPTURE" ] && [ -n "$prompt" ]; then
  cp "$prompt" "$DEVIN_PROMPT_CAPTURE"
fi
case "$DEVIN_MODE" in
  exit)
    echo 'synthetic devin stderr' >&2
    exit 7
    ;;
  *)
    echo 'session: brisk-otter'
    echo '4'
    ;;
esac
`
}

func newFakeDevinBackend(t *testing.T, env map[string]string) *devinBackend {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	path := filepath.Join(t.TempDir(), "devin")
	writeTestExecutable(t, path, []byte(fakeDevinScript()))
	return &devinBackend{cfg: Config{ExecutablePath: path, Logger: slog.Default(), Env: env}}
}

func awaitDevinResult(t *testing.T, session *Session) ([]Message, Result) {
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
		t.Fatal("timed out waiting for devin result")
		return nil, Result{}
	}
}

func TestDevinBackendPrintsPromptFileNotArgv(t *testing.T) {
	t.Parallel()
	argsFile := filepath.Join(t.TempDir(), "args")
	promptFile := filepath.Join(t.TempDir(), "prompt")
	backend := newFakeDevinBackend(t, map[string]string{
		"DEVIN_ARGS_FILE":      argsFile,
		"DEVIN_PROMPT_CAPTURE": promptFile,
	})
	session, err := backend.Execute(context.Background(), "what is 2+2?", ExecOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	messages, result := awaitDevinResult(t, session)
	if result.Status != "completed" || result.SessionID != devinFixtureSession {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !strings.Contains(result.Output, "4") {
		t.Fatalf("output missing printed answer: %q", result.Output)
	}
	var sawText bool
	for _, message := range messages {
		if message.Type == MessageText && strings.Contains(message.Content, "4") {
			sawText = true
		}
	}
	if !sawText {
		t.Fatalf("expected MessageText with printed output, got %#v", messages)
	}
	argv, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	if strings.Contains(string(argv), "what is 2+2?") {
		t.Fatalf("prompt leaked onto argv: %s", argv)
	}
	if !strings.Contains(string(argv), "--print") || !strings.Contains(string(argv), "--prompt-file") {
		t.Fatalf("managed flags missing from argv: %s", argv)
	}
	gotPrompt, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("read captured prompt: %v", err)
	}
	if string(gotPrompt) != "what is 2+2?" {
		t.Fatalf("prompt file = %q", gotPrompt)
	}
}

func TestDevinBackendResumeArgv(t *testing.T) {
	t.Parallel()
	argsFile := filepath.Join(t.TempDir(), "args")
	backend := newFakeDevinBackend(t, map[string]string{"DEVIN_ARGS_FILE": argsFile})
	session, err := backend.Execute(context.Background(), "continue", ExecOptions{
		Timeout:         5 * time.Second,
		ResumeSessionID: devinFixtureSession,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	_, result := awaitDevinResult(t, session)
	if result.Status != "completed" || result.SessionID != devinFixtureSession {
		t.Fatalf("unexpected result: %+v", result)
	}
	argv, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	joined := strings.ReplaceAll(string(argv), "\n", " ")
	if !strings.Contains(joined, "--resume "+devinFixtureSession) {
		t.Fatalf("resume id missing from argv: %s", argv)
	}
	if strings.Contains(joined, "-c") || strings.Contains(string(argv), "--continue") {
		t.Fatalf("bare continue leaked: %s", argv)
	}
}

func TestDevinBackendRejectsBadResumeBeforeSpawn(t *testing.T) {
	t.Parallel()
	backend := newFakeDevinBackend(t, nil)
	if _, err := backend.Execute(context.Background(), "hi", ExecOptions{ResumeSessionID: "devin-abc123"}); err == nil {
		t.Fatal("cloud-shaped resume must fail before spawn")
	}
	if _, err := backend.Execute(context.Background(), "hi", ExecOptions{ResumeSessionID: "--resume"}); err == nil {
		t.Fatal("flag-like resume must fail before spawn")
	}
}

func TestDevinBackendFailedExit(t *testing.T) {
	t.Parallel()
	backend := newFakeDevinBackend(t, map[string]string{"DEVIN_MODE": "exit"})
	session, err := backend.Execute(context.Background(), "fail", ExecOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	_, result := awaitDevinResult(t, session)
	if result.Status != "failed" {
		t.Fatalf("status = %q, want failed", result.Status)
	}
	if result.SessionID != "" {
		t.Fatalf("failed run must not persist a session id, got %q", result.SessionID)
	}
}

const (
	devinModelsArrayJSON    = `[{"id":"opus","name":"Claude Opus","default":true},{"id":"claude-sonnet-4","name":"Claude Sonnet 4"}]`
	devinModelsFamiliesJSON = `{"families":[{"family":"Anthropic","models":[{"slug":"opus","id":"claude-opus-4.6","cost_summary":"$x"},{"id":"sonnet","name":"Sonnet"}]}]}`
	devinModelsWrappedJSON  = `{"models":[{"model_id":"swe-1-6-fast","display_name":"SWE-1.6 Fast"}]}`
)

func TestModelSelectionSupportedDevin(t *testing.T) {
	t.Parallel()
	if !ModelSelectionSupported("devin") {
		t.Fatal("ModelSelectionSupported(devin) must be true so the picker sends --model")
	}
}

func TestListModelsDevin(t *testing.T) {
	t.Parallel()

	t.Run("array catalog", func(t *testing.T) {
		t.Parallel()
		path, argsFile := fakeDevinModelsCLI(t, devinModelsArrayJSON, 0)
		cat, err := ListModels(context.Background(), "devin", Command{Path: path})
		if err != nil {
			t.Fatalf("ListModels: %v", err)
		}
		assertDevinModelsArgv(t, argsFile)
		if cat.Fallback {
			t.Fatal("live catalog must not be marked Fallback")
		}
		if len(cat.Models) != 2 {
			t.Fatalf("models = %#v, want opus and claude-sonnet-4", cat.Models)
		}
		if cat.Models[0].ID != "opus" || !cat.Models[0].Default || cat.Models[0].Label != "Claude Opus" || cat.Models[0].Provider != "devin" {
			t.Fatalf("first model = %#v, want opus default Claude Opus / devin", cat.Models[0])
		}
		if cat.Models[1].ID != "claude-sonnet-4" || cat.Models[1].Default || cat.Models[1].Label != "Claude Sonnet 4" {
			t.Fatalf("second model = %#v, want claude-sonnet-4", cat.Models[1])
		}
	})

	t.Run("families prefers id over slug", func(t *testing.T) {
		t.Parallel()
		path, argsFile := fakeDevinModelsCLI(t, devinModelsFamiliesJSON, 0)
		cat, err := ListModels(context.Background(), "devin", Command{Path: path})
		if err != nil {
			t.Fatalf("ListModels: %v", err)
		}
		assertDevinModelsArgv(t, argsFile)
		if len(cat.Models) != 2 {
			t.Fatalf("models = %#v, want claude-opus-4.6 and sonnet", cat.Models)
		}
		if cat.Models[0].ID != "claude-opus-4.6" {
			t.Fatalf("id = %q, want claude-opus-4.6 not slug opus", cat.Models[0].ID)
		}
		if cat.Models[0].Provider != "Anthropic" || cat.Models[1].Provider != "Anthropic" {
			t.Fatalf("provider = (%q, %q), want Anthropic from family", cat.Models[0].Provider, cat.Models[1].Provider)
		}
		if cat.Models[0].Label != "claude-opus-4.6" {
			t.Fatalf("label = %q, cost_summary must not become the label", cat.Models[0].Label)
		}
		if cat.Models[1].ID != "sonnet" || cat.Models[1].Label != "Sonnet" {
			t.Fatalf("second model = %#v", cat.Models[1])
		}
	})

	t.Run("wrapped model_id", func(t *testing.T) {
		t.Parallel()
		path, argsFile := fakeDevinModelsCLI(t, devinModelsWrappedJSON, 0)
		cat, err := ListModels(context.Background(), "devin", Command{Path: path})
		if err != nil {
			t.Fatalf("ListModels: %v", err)
		}
		assertDevinModelsArgv(t, argsFile)
		if len(cat.Models) != 1 || cat.Models[0].ID != "swe-1-6-fast" || cat.Models[0].Label != "SWE-1.6 Fast" || cat.Models[0].Provider != "devin" {
			t.Fatalf("models = %#v, want swe-1-6-fast", cat.Models)
		}
	})

	t.Run("missing binary", func(t *testing.T) {
		t.Parallel()
		cat, err := ListModels(context.Background(), "devin", Command{Path: "/nonexistent/devin"})
		if err != nil {
			t.Fatalf("ListModels: %v", err)
		}
		if len(cat.Models) != 0 || !cat.Fallback {
			t.Fatalf("missing binary catalog = %#v fallback=%v, want empty Fallback", cat.Models, cat.Fallback)
		}
	})

	t.Run("failing binary", func(t *testing.T) {
		t.Parallel()
		path, argsFile := fakeDevinModelsCLI(t, "", 1)
		cat, err := ListModels(context.Background(), "devin", Command{Path: path})
		if err != nil {
			t.Fatalf("ListModels: %v", err)
		}
		assertDevinModelsArgv(t, argsFile)
		if len(cat.Models) != 0 || !cat.Fallback {
			t.Fatalf("failing binary catalog = %#v fallback=%v, want empty Fallback", cat.Models, cat.Fallback)
		}
	})
}

func TestParseDevinModelsJSON(t *testing.T) {
	t.Parallel()

	t.Run("garbage and empty", func(t *testing.T) {
		t.Parallel()
		for _, raw := range []string{"", "{}", "[]", "null", `{"foo":1}`, `{"models":[]}`} {
			if got := parseDevinModelsJSON([]byte(raw)); len(got) != 0 {
				t.Fatalf("parseDevinModelsJSON(%q) = %#v, want empty", raw, got)
			}
		}
	})

	t.Run("text blob", func(t *testing.T) {
		t.Parallel()
		if got := parseDevinModelsJSON([]byte("Available models\nopus\nclaude-sonnet-4\n")); len(got) != 0 {
			t.Fatalf("non-JSON blob = %#v, want empty", got)
		}
	})

	t.Run("nested and official ids", func(t *testing.T) {
		t.Parallel()
		raw := `{"data":{"items":[{"id":"claude-sonnet-4"},{"slug":"claude-opus-4.6"},{"model":"opus"},{"modelId":"codex"},{"model_id":"swe"},{"id":"--model"},{"id":"/tmp/opus"},{"id":"opus"}]}}`
		got := parseDevinModelsJSON([]byte(raw))
		want := []string{"claude-sonnet-4", "claude-opus-4.6", "opus", "codex", "swe"}
		if len(got) != len(want) {
			t.Fatalf("ids = %#v, want %v", got, want)
		}
		for i, id := range want {
			if got[i].ID != id {
				t.Fatalf("got[%d].ID = %q, want %q", i, got[i].ID, id)
			}
		}
	})

	t.Run("first id wins on dedup", func(t *testing.T) {
		t.Parallel()
		got := parseDevinModelsJSON([]byte(`[{"id":"opus","name":"First"},{"id":"opus","name":"Second"}]`))
		if len(got) != 1 || got[0].Label != "First" {
			t.Fatalf("dedup = %#v, want first label", got)
		}
	})
}

func fakeDevinModelsCLI(t *testing.T, stdout string, exitCode int) (path, argsFile string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	dir := t.TempDir()
	path = filepath.Join(dir, "devin")
	argsFile = filepath.Join(dir, "args")
	jsonFile := filepath.Join(dir, "models.json")
	if err := os.WriteFile(jsonFile, []byte(stdout), 0o600); err != nil {
		t.Fatalf("write models.json: %v", err)
	}
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" > \"" + argsFile + "\"\n" +
		"if [ \"$1\" != \"models\" ] || [ \"$2\" != \"list\" ] || [ \"$3\" != \"--format\" ] || [ \"$4\" != \"json\" ]; then\n" +
		"  exit 1\n" +
		"fi\n" +
		"cat \"" + jsonFile + "\"\n" +
		"exit " + strconv.Itoa(exitCode) + "\n"
	writeTestExecutable(t, path, []byte(script))
	return path, argsFile
}

func assertDevinModelsArgv(t *testing.T, argsFile string) {
	t.Helper()
	raw, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read captured argv: %v", err)
	}
	got := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	want := []string{"models", "list", "--format", "json"}
	if len(got) != len(want) {
		t.Fatalf("argv = %q, want %v", raw, want)
	}
	for i, token := range want {
		if got[i] != token {
			t.Fatalf("argv[%d] = %q, want %q (full %q)", i, got[i], token, raw)
		}
	}
}
