package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ampBackend drives Amp's headless execute protocol.
// Fresh:  amp --execute --stream-json --stream-json-thinking --dangerously-allow-all
// Resume: amp threads continue <T-uuid> --execute --stream-json …
// The prompt is piped on stdin so user-influenced text never rides argv
// (same class of fix as cursor #5649 and qwen #6082).
type ampBackend struct {
	cfg Config
}

// ampThreadID is a validated Amp thread id ("T-" + UUID). The zero value
// means a fresh run. The only constructors are parseAmpThreadID and
// resolveAmpResume, so an unvalidated string cannot reach the resume argv
// or Result.SessionID.
type ampThreadID string

var ampThreadIDPattern = regexp.MustCompile(`(?i)^T-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func parseAmpThreadID(s string) (ampThreadID, bool) {
	if !ampThreadIDPattern.MatchString(s) {
		return "", false
	}
	return ampThreadID(s), true
}

func resolveAmpResume(resumeSessionID string) (ampThreadID, error) {
	if resumeSessionID == "" {
		return "", nil
	}
	id, ok := parseAmpThreadID(resumeSessionID)
	if !ok {
		return "", fmt.Errorf("amp resume session %q is not a T-<uuid> thread id", resumeSessionID)
	}
	return id, nil
}

// ampBlockedArgs are daemon-owned. threads/continue are stripped as a
// sequence by filterAmpThreadSequences so a leftover T- id cannot become a
// stray positional.
var ampBlockedArgs = map[string]blockedArgMode{
	"-x":                      blockedStandalone,
	"--execute":               blockedStandalone,
	"--stream-json":           blockedStandalone,
	"--stream-json-input":     blockedStandalone,
	"--stream-json-thinking":  blockedStandalone,
	"--dangerously-allow-all": blockedStandalone,
	"--mcp-config":            blockedWithValue,
	"--resume":                blockedWithValue,
	"--continue":              blockedStandalone,
	"--no-tui":                blockedStandalone,
	"--executor":              blockedWithValue,
	"-p":                      blockedWithValue,
	"--output-format":         blockedWithValue,
}

func filterAmpRuntimeArgs(args []string, logger *slog.Logger) []string {
	return filterAmpThreadSequences(filterCustomArgs(args, ampBlockedArgs, logger), logger)
}

// filterAmpThreadSequences removes `threads continue [<T-uuid>]` as a unit.
// Stripping the verbs one token at a time would leave a stray thread id on
// argv, which Amp would treat as a positional.
func filterAmpThreadSequences(args []string, logger *slog.Logger) []string {
	if len(args) == 0 {
		return args
	}
	if logger == nil {
		logger = slog.Default()
	}
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "threads" && i+1 < len(args) && args[i+1] == "continue" {
			logger.Warn("custom_args: blocked amp thread-continue sequence, skipping")
			i++
			if i+1 < len(args) {
				if _, ok := parseAmpThreadID(args[i+1]); ok {
					i++
				}
			}
			continue
		}
		if arg == "threads" {
			logger.Warn("custom_args: blocked amp threads token, skipping")
			continue
		}
		out = append(out, arg)
	}
	return out
}

func buildAmpArgs(resume ampThreadID, opts ExecOptions, logger *slog.Logger) []string {
	var args []string
	if resume != "" {
		args = append(args, "threads", "continue", string(resume))
	}
	args = append(args, "--execute", "--stream-json", "--stream-json-thinking", "--dangerously-allow-all")
	args = append(args, filterAmpRuntimeArgs(opts.ExtraArgs, logger)...)
	args = append(args, filterAmpRuntimeArgs(opts.CustomArgs, logger)...)
	return args
}

func (b *ampBackend) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error) {
	resume, err := resolveAmpResume(opts.ResumeSessionID)
	if err != nil {
		return nil, err
	}
	execName := b.cfg.ExecutablePath
	if execName == "" {
		execName = "amp"
	}
	lookedUp, err := exec.LookPath(execName)
	if err != nil {
		return nil, fmt.Errorf("amp executable not found at %q: %w", execName, err)
	}
	timeout := opts.Timeout
	runCtx, cancel := runContext(ctx, timeout)
	if opts.Model != "" {
		b.cfg.Logger.Debug("amp ignores ExecOptions.Model; Amp selects its own model", "model", opts.Model)
	}
	if opts.ThinkingLevel != "" {
		b.cfg.Logger.Debug("amp ignores ExecOptions.ThinkingLevel", "thinking_level", opts.ThinkingLevel)
	}
	if hasManagedMcpConfig(opts.McpConfig) {
		b.cfg.Logger.Debug("amp ignores ExecOptions.McpConfig until --mcp-config file-path support is verified")
	}
	args := buildAmpArgs(resume, opts, b.cfg.Logger)
	cmd, _, _ := b.cfg.commandAt(execName).execVia(runCtx, chooseAmpInvocation, lookedUp, args, b.cfg.Logger)
	hideAgentWindow(cmd)
	b.cfg.logAgentCommand(cmd, newAgentCommandLogArgs(args))
	cmd.WaitDelay = 10 * time.Second
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}
	cmd.Env = buildEnv(b.cfg.Env)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("amp stdout pipe: %w", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("amp stdin pipe: %w", err)
	}
	var closeStdinOnce sync.Once
	closeStdin := func() { closeStdinOnce.Do(func() { _ = stdin.Close() }) }
	stderrBuf := newStderrTail(newLogWriter(b.cfg.Logger, "[amp:stderr] "), agentStderrTailBytes)
	cmd.Stderr = stderrBuf
	if err := startOwnedProcessTree(cmd, b.cfg.Logger); err != nil {
		closeStdin()
		cancel()
		return nil, fmt.Errorf("start amp: %w", err)
	}
	b.cfg.Logger.Info("amp started", "pid", cmd.Process.Pid, "cwd", opts.Cwd)

	writeErrCh := make(chan error, 1)
	go func() {
		_, err := io.WriteString(stdin, prompt)
		closeStdin()
		writeErrCh <- err
	}()

	msgCh := make(chan Message, 256)
	resCh := make(chan Result, 1)
	go func() {
		defer cancel()
		defer close(msgCh)
		defer close(resCh)

		started := time.Now()
		state := ampStreamState{usage: make(map[string]TokenUsage)}
		go func() {
			<-runCtx.Done()
			closeStdin()
			_ = stdout.Close()
		}()

		scanner := newAgentStreamScanner(stdout)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var event ampStreamEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				state.invalidEventCount++
				continue
			}
			state.eventCount++
			state.lastEventType = event.Type
			handleAmpEvent(event, msgCh, &state)
		}
		scanErr := scanner.Err()
		if scanErr != nil {
			_ = stdout.Close()
		}
		exitErr := cmd.Wait()
		releaseProcessGroup(cmd)
		duration := time.Since(started)
		writeErr := <-writeErrCh

		emitted := string(state.threadID)
		status, output, errMsg := finalizeStreamResult("amp", timeout, runCtx.Err(), writeErr, exitErr, emitted, streamTerminalState{
			lastAssistantText: state.lastAssistantText,
			finalResultText:   state.finalResultText,
			sawResult:         state.sawResult,
			resultIsError:     state.resultIsError,
			scanErr:           scanErr,
		}, "")
		if errMsg != "" {
			errMsg = withAgentStderr(errMsg, "amp", stderrBuf.Tail())
		}
		logStreamProtocolObservation(b.cfg.Logger, streamProtocolObservation{
			provider: "amp", cliVersion: b.cfg.CLIVersion, model: state.model,
			exitCode: streamProcessExitCode(exitErr), eventCount: state.eventCount,
			invalidEventCount: state.invalidEventCount, assistantEventCount: state.assistantEventCount,
			toolUseCount: state.toolUseCount, sawResult: state.sawResult, resultIsError: state.resultIsError,
			resultBytes: len(state.finalResultText), lastAssistantBytes: len(state.lastAssistantText),
			scannerError: scanErr != nil, lastEventType: state.lastEventType,
			unreadableAssistantCount: state.unreadableAssistantCount,
		})
		b.cfg.Logger.Info("amp finished", "pid", cmd.Process.Pid, "status", status, "duration", duration.Round(time.Millisecond).String())
		resCh <- Result{
			Status: status, Output: output, Error: errMsg, DurationMs: duration.Milliseconds(),
			SessionID:      resolveSessionID(string(resume), emitted, status == "failed", errMsg),
			Usage:          state.usage,
			ResumeRejected: resumeWasRejected(string(resume), emitted, status == "failed", errMsg),
		}
	}()
	return &Session{Messages: msgCh, Result: resCh}, nil
}

type ampStreamEvent struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Message   json.RawMessage `json:"message,omitempty"`
	Result    string          `json:"result,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
	Usage     *ampUsage       `json:"usage,omitempty"`
	Error     json.RawMessage `json:"error,omitempty"`
}

type ampMessage struct {
	Model   string            `json:"model,omitempty"`
	Content []ampContentBlock `json:"content"`
	Usage   *ampUsage         `json:"usage,omitempty"`
}

type ampUsage struct {
	InputTokens          int64 `json:"input_tokens"`
	OutputTokens         int64 `json:"output_tokens"`
	CacheReadInputTokens int64 `json:"cache_read_input_tokens"`
}

type ampContentBlock struct {
	Type      string          `json:"type"`
	Thinking  string          `json:"thinking,omitempty"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
}

type ampStreamState struct {
	threadID                                                         ampThreadID
	model, lastAssistantText, finalResultText, lastEventType         string
	sawResult, resultIsError                                         bool
	usage                                                            map[string]TokenUsage
	eventCount, invalidEventCount, assistantEventCount, toolUseCount int
	unreadableAssistantCount                                         int
}

func handleAmpEvent(event ampStreamEvent, ch chan<- Message, state *ampStreamState) {
	if id, ok := parseAmpThreadID(event.SessionID); ok {
		state.threadID = id
	}
	switch event.Type {
	case "system":
		trySend(ch, Message{Type: MessageStatus, Status: "running", SessionID: string(state.threadID)})
	case "assistant":
		state.assistantEventCount++
		turn, model := handleAmpAssistant(event.Message, ch, state.usage)
		if model != "" {
			state.model = model
		}
		state.toolUseCount += turn.toolUses
		if !turn.understood {
			state.unreadableAssistantCount++
		}
		state.lastAssistantText = turn.resolveFallback(state.lastAssistantText)
	case "user":
		handleAmpUser(event.Message, ch)
	case "result":
		state.sawResult = true
		state.resultIsError = ampResultIsError(event.Subtype, event.IsError)
		if state.resultIsError {
			state.finalResultText = ampErrorText(event)
		} else {
			state.finalResultText = event.Result
		}
		if usage := ampResultUsage(event.Usage, state.model); len(usage) > 0 {
			state.usage = usage
		}
	default:
		if event.Type == "error" || strings.HasPrefix(event.Type, "error") {
			state.sawResult = true
			state.resultIsError = true
			state.finalResultText = ampErrorText(event)
		}
	}
}

func handleAmpAssistant(raw json.RawMessage, ch chan<- Message, usage map[string]TokenUsage) (assistantTurn, string) {
	var message ampMessage
	if json.Unmarshal(raw, &message) != nil {
		return assistantTurn{}, ""
	}
	turn := assistantTurn{understood: true}
	if message.Usage != nil && message.Model != "" {
		usage[message.Model] = ampTokenUsage(message.Usage)
	}
	var text strings.Builder
	tools := 0
	for _, block := range message.Content {
		switch block.Type {
		case "thinking":
			content := block.Thinking
			if content == "" {
				content = block.Text
			}
			if content != "" {
				trySend(ch, Message{Type: MessageThinking, Content: content})
			}
		case "redacted_thinking":
			// Understood, no text. Falling through would clear the #6006 fallback.
		case "text":
			if block.Text != "" {
				text.WriteString(block.Text)
				trySend(ch, Message{Type: MessageText, Content: block.Text})
			}
		case "tool_use":
			tools++
			var input map[string]any
			if len(block.Input) > 0 {
				_ = json.Unmarshal(block.Input, &input)
			}
			trySend(ch, Message{Type: MessageToolUse, Tool: block.Name, CallID: block.ID, Input: input})
		default:
			turn.understood = false
		}
	}
	turn.text = text.String()
	turn.toolUses = tools
	return turn, message.Model
}

func handleAmpUser(raw json.RawMessage, ch chan<- Message) {
	var message ampMessage
	if json.Unmarshal(raw, &message) != nil {
		return
	}
	for _, block := range message.Content {
		if block.Type == "tool_result" {
			trySend(ch, Message{Type: MessageToolResult, CallID: block.ToolUseID, Output: ampToolResultOutput(block.Content)})
		}
	}
}

func ampTokenUsage(usage *ampUsage) TokenUsage {
	return TokenUsage{InputTokens: usage.InputTokens, OutputTokens: usage.OutputTokens, CacheReadTokens: usage.CacheReadInputTokens}
}

func ampResultUsage(usage *ampUsage, model string) map[string]TokenUsage {
	if usage == nil || model == "" || (usage.InputTokens == 0 && usage.OutputTokens == 0 && usage.CacheReadInputTokens == 0) {
		return nil
	}
	return map[string]TokenUsage{model: ampTokenUsage(usage)}
}

func ampToolResultOutput(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	return string(raw)
}

func ampResultIsError(subtype string, isError bool) bool {
	return isError || subtype != "success"
}

func ampErrorText(event ampStreamEvent) string {
	if event.Result != "" {
		return event.Result
	}
	var body struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(event.Error, &body) == nil && body.Message != "" {
		return body.Message
	}
	if len(event.Error) > 0 {
		return string(event.Error)
	}
	return "amp returned an error event without details"
}
