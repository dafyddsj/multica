package agent

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// devinBackend drives Cognition's local Devin CLI.
// Fresh:  devin --print --prompt-file <tmp> --permission-mode dangerous --respect-workspace-trust false
// Resume: the same flags plus --resume <parsed-id>
//
// The prompt lives in a 0600 temp file. Official help also allows -p "prompt";
// that form is never used because user text on argv breaks on Windows the same
// way Qwen and Cursor already did.
type devinBackend struct {
	cfg Config
}

// devinSessionID is a validated local CLI session id. The zero value means a
// fresh run. The only constructors are parseDevinSessionID and
// resolveDevinResume, so an unvalidated string cannot reach resume argv or
// Result.SessionID.
type devinSessionID string

// Official docs show brisk-otter and abc12345. Cloud DRS box ids look like
// devin-abc123 and are a different token until a canary proves otherwise.
var devinSessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,64}$`)

var devinSessionLabelPattern = regexp.MustCompile(`(?i)(?:session(?:[ _-]?id)?|resum(?:e|ing))\s*[:=]\s*([A-Za-z0-9][A-Za-z0-9_-]{1,64})`)

func parseDevinSessionID(s string) (devinSessionID, bool) {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "-") {
		return "", false
	}
	if strings.ContainsAny(s, "/\\ \t\n") || strings.Contains(s, "://") {
		return "", false
	}
	if strings.HasPrefix(strings.ToLower(s), "devin-") {
		return "", false
	}
	if isAllDigits(s) {
		return "", false
	}
	if !devinSessionIDPattern.MatchString(s) {
		return "", false
	}
	return devinSessionID(s), true
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func resolveDevinResume(resumeSessionID string) (devinSessionID, error) {
	if resumeSessionID == "" {
		return "", nil
	}
	id, ok := parseDevinSessionID(resumeSessionID)
	if !ok {
		return "", fmt.Errorf("devin resume session %q is not a local CLI session id", resumeSessionID)
	}
	return id, nil
}

func extractDevinSessionID(text string) string {
	var last string
	for _, match := range devinSessionLabelPattern.FindAllStringSubmatch(text, -1) {
		if len(match) < 2 {
			continue
		}
		if id, ok := parseDevinSessionID(match[1]); ok {
			last = string(id)
		}
	}
	return last
}

var devinBlockedArgs = map[string]blockedArgMode{
	"-p":                        blockedOptionalValue,
	"--print":                   blockedOptionalValue,
	"--prompt-file":             blockedWithValue,
	"--permission-mode":         blockedWithValue,
	"--respect-workspace-trust": blockedOptionalValue,
	"--model":                   blockedWithValue,
	"-c":                        blockedStandalone,
	"--continue":                blockedStandalone,
	"-r":                        blockedOptionalValue,
	"--resume":                  blockedOptionalValue,
	"--sandbox":                 blockedStandalone,
}

var devinBlockedCommands = map[string]bool{
	"acp":     true,
	"cloud":   true,
	"ssh":     true,
	"desktop": true,
}

func filterDevinRuntimeArgs(args []string, logger *slog.Logger) []string {
	return filterDevinCommands(filterCustomArgs(args, devinBlockedArgs, logger), logger)
}

func filterDevinCommands(args []string, logger *slog.Logger) []string {
	if len(args) == 0 {
		return args
	}
	if logger == nil {
		logger = slog.Default()
	}
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if devinBlockedCommands[arg] {
			logger.Warn("custom_args: blocked devin subcommand, skipping", "arg", arg)
			continue
		}
		out = append(out, arg)
	}
	return out
}

func buildDevinArgs(resume devinSessionID, opts ExecOptions, promptFile string, logger *slog.Logger) []string {
	args := []string{
		"--print",
		"--prompt-file", promptFile,
		"--permission-mode", "dangerous",
		"--respect-workspace-trust", "false",
	}
	if resume != "" {
		args = append(args, "--resume", string(resume))
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	args = append(args, filterDevinRuntimeArgs(opts.ExtraArgs, logger)...)
	args = append(args, filterDevinRuntimeArgs(opts.CustomArgs, logger)...)
	return args
}

func writeDevinPromptFile(prompt string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "multica-devin-prompt-*")
	if err != nil {
		return "", nil, fmt.Errorf("create devin prompt temp dir: %w", err)
	}
	path := filepath.Join(dir, "prompt.txt")
	if err := os.WriteFile(path, []byte(prompt), 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, fmt.Errorf("write devin prompt file: %w", err)
	}
	return path, func() { _ = os.RemoveAll(dir) }, nil
}

func chooseDevinInvocation(execName, _ string, args []string, _ *slog.Logger) (string, []string) {
	return execName, args
}

func (b *devinBackend) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error) {
	resume, err := resolveDevinResume(opts.ResumeSessionID)
	if err != nil {
		return nil, err
	}
	execName := b.cfg.ExecutablePath
	if execName == "" {
		execName = "devin"
	}
	lookedUp, err := exec.LookPath(execName)
	if err != nil {
		return nil, fmt.Errorf("devin executable not found at %q: %w", execName, err)
	}
	promptPath, cleanupPrompt, err := writeDevinPromptFile(prompt)
	if err != nil {
		return nil, err
	}
	timeout := opts.Timeout
	runCtx, cancel := runContext(ctx, timeout)
	if opts.ThinkingLevel != "" {
		b.cfg.Logger.Debug("devin ignores ExecOptions.ThinkingLevel", "thinking_level", opts.ThinkingLevel)
	}
	args := buildDevinArgs(resume, opts, promptPath, b.cfg.Logger)

	cmd, _, _ := b.cfg.commandAt(execName).execVia(runCtx, chooseDevinInvocation, lookedUp, args, b.cfg.Logger)
	hideAgentWindow(cmd)
	b.cfg.logAgentCommand(cmd, newAgentCommandLogArgs(args))
	cmd.WaitDelay = 10 * time.Second
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}
	cmd.Env = buildEnv(b.cfg.Env)

	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	stderrBuf := newStderrTail(newLogWriter(b.cfg.Logger, "[devin:stderr] "), agentStderrTailBytes)
	cmd.Stderr = stderrBuf
	if err := startOwnedProcessTree(cmd, b.cfg.Logger); err != nil {
		cleanupPrompt()
		cancel()
		return nil, fmt.Errorf("start devin: %w", err)
	}
	b.cfg.Logger.Info("devin started", "pid", cmd.Process.Pid, "cwd", opts.Cwd, "model", opts.Model)

	msgCh := make(chan Message, 256)
	resCh := make(chan Result, 1)
	go func() {
		defer cancel()
		defer close(msgCh)
		defer close(resCh)
		defer cleanupPrompt()

		started := time.Now()
		exitErr := cmd.Wait()
		releaseProcessGroup(cmd)
		duration := time.Since(started)
		output := strings.TrimRightFunc(stdoutBuf.String(), unicode.IsSpace)
		status := "completed"
		errMsg := ""
		if runCtx.Err() != nil {
			status = "failed"
			if timeout > 0 && ctx.Err() == nil {
				errMsg = fmt.Sprintf("devin timed out after %s", timeout)
			} else {
				errMsg = "devin cancelled"
			}
		} else if exitErr != nil {
			status = "failed"
			errMsg = exitErr.Error()
		}
		emitted := extractDevinSessionID(output)
		if emitted == "" && resume != "" && status != "failed" {
			emitted = string(resume)
		}
		if errMsg != "" {
			errMsg = withAgentStderr(errMsg, "devin", stderrBuf.Tail())
		}
		if output != "" {
			trySend(msgCh, Message{Type: MessageText, Content: output})
		}
		b.cfg.Logger.Info("devin finished", "pid", cmd.Process.Pid, "status", status, "duration", duration.Round(time.Millisecond).String())
		resCh <- Result{
			Status:         status,
			Output:         output,
			Error:          errMsg,
			DurationMs:     duration.Milliseconds(),
			SessionID:      resolveSessionID(string(resume), emitted, status == "failed", errMsg),
			ResumeRejected: resumeWasRejected(string(resume), emitted, status == "failed", errMsg),
		}
	}()
	return &Session{Messages: msgCh, Result: resCh}, nil
}
