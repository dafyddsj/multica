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

// gooseBackend drives Goose's headless run protocol.
// Fresh:  goose run -i - --output-format stream-json
// Resume: goose run -i - --output-format stream-json --resume --session-id <id>
//
// The prompt is piped on stdin through -i - so user-influenced text never
// rides argv (same class of fix as cursor #5649, qwen #6082, and Amp stdin).
type gooseBackend struct {
	cfg Config
}

// gooseSessionID is a validated Goose session id. The zero value means a
// fresh run. The only constructors are parseGooseSessionID and
// resolveGooseResume, so an unvalidated string cannot reach the resume argv
// or Result.SessionID.
type gooseSessionID string

// Documented Goose 1.48.0 shapes: 20250325_200615 (--path help) and
// 20251108_1 (CLI commands guide). Fail closed on anything else.
var gooseSessionIDPattern = regexp.MustCompile(`^[0-9]{8}_[A-Za-z0-9]+$`)

func parseGooseSessionID(s string) (gooseSessionID, bool) {
	if !gooseSessionIDPattern.MatchString(s) {
		return "", false
	}
	return gooseSessionID(s), true
}

func resolveGooseResume(resumeSessionID string) (gooseSessionID, error) {
	if resumeSessionID == "" {
		return "", nil
	}
	id, ok := parseGooseSessionID(resumeSessionID)
	if !ok {
		return "", fmt.Errorf("goose resume session %q is not a YYYYMMDD_<id> session id", resumeSessionID)
	}
	return id, nil
}

var gooseBlockedArgs = map[string]blockedArgMode{
	"-i":              blockedWithValue,
	"--instructions":  blockedWithValue,
	"-t":              blockedWithValue,
	"--text":          blockedWithValue,
	"--output-format": blockedWithValue,
	"--resume":        blockedStandalone,
	"-r":              blockedStandalone,
	"--session-id":    blockedWithValue,
	"--no-session":    blockedStandalone,
	"-s":              blockedStandalone,
	"--interactive":   blockedStandalone,
	"--recipe":        blockedWithValue,
	"--acp":           blockedStandalone,
	"acp":             blockedStandalone,
	"serve":           blockedStandalone,
}

func filterGooseRuntimeArgs(args []string, logger *slog.Logger) []string {
	return filterCustomArgs(args, gooseBlockedArgs, logger)
}

func buildGooseArgs(resume gooseSessionID, opts ExecOptions, logger *slog.Logger) []string {
	args := []string{"run", "-i", "-", "--output-format", "stream-json"}
	if resume != "" {
		args = append(args, "--resume", "--session-id", string(resume))
	}
	if model := strings.TrimSpace(opts.Model); model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, filterGooseRuntimeArgs(opts.ExtraArgs, logger)...)
	args = append(args, filterGooseRuntimeArgs(opts.CustomArgs, logger)...)
	return args
}

func writeGooseInput(w io.Writer, prompt string) error {
	_, err := io.WriteString(w, prompt)
	return err
}

func chooseGooseInvocation(execName, _ string, args []string, _ *slog.Logger) (string, []string) {
	return execName, args
}

func gooseProcessEnv(extra map[string]string) []string {
	merged := make(map[string]string, len(extra)+1)
	for k, v := range extra {
		merged[k] = v
	}
	merged["GOOSE_MODE"] = "auto"
	return buildEnv(merged)
}

func (b *gooseBackend) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error) {
	resume, err := resolveGooseResume(opts.ResumeSessionID)
	if err != nil {
		return nil, err
	}
	execName := b.cfg.ExecutablePath
	if execName == "" {
		execName = "goose"
	}
	lookedUp, err := exec.LookPath(execName)
	if err != nil {
		return nil, fmt.Errorf("goose executable not found at %q: %w", execName, err)
	}
	timeout := opts.Timeout
	runCtx, cancel := runContext(ctx, timeout)
	if opts.ThinkingLevel != "" {
		b.cfg.Logger.Debug("goose ignores ExecOptions.ThinkingLevel", "thinking_level", opts.ThinkingLevel)
	}
	args := buildGooseArgs(resume, opts, b.cfg.Logger)

	cmd, _, _ := b.cfg.commandAt(execName).execVia(runCtx, chooseGooseInvocation, lookedUp, args, b.cfg.Logger)
	hideAgentWindow(cmd)
	b.cfg.logAgentCommand(cmd, newAgentCommandLogArgs(args))
	cmd.WaitDelay = 10 * time.Second
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}
	cmd.Env = gooseProcessEnv(b.cfg.Env)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("goose stdout pipe: %w", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("goose stdin pipe: %w", err)
	}
	var closeStdinOnce sync.Once
	closeStdin := func() { closeStdinOnce.Do(func() { _ = stdin.Close() }) }
	stderrBuf := newStderrTail(newLogWriter(b.cfg.Logger, "[goose:stderr] "), agentStderrTailBytes)
	cmd.Stderr = stderrBuf
	if err := startOwnedProcessTree(cmd, b.cfg.Logger); err != nil {
		closeStdin()
		cancel()
		return nil, fmt.Errorf("start goose: %w", err)
	}
	b.cfg.Logger.Info("goose started", "pid", cmd.Process.Pid, "cwd", opts.Cwd, "model", opts.Model)

	writeErrCh := make(chan error, 1)
	go func() {
		err := writeGooseInput(stdin, prompt)
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
		state := gooseStreamState{usage: make(map[string]TokenUsage)}
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
			var event gooseStreamEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				state.invalidEventCount++
				continue
			}
			state.eventCount++
			state.lastEventType = event.Type
			handleGooseEvent(event, msgCh, &state)
		}
		scanErr := scanner.Err()
		if scanErr != nil {
			_ = stdout.Close()
		}
		exitErr := cmd.Wait()
		releaseProcessGroup(cmd)
		duration := time.Since(started)
		writeErr := <-writeErrCh

		emitted := string(state.sessionID)
		if emitted == "" && resume != "" {
			emitted = string(resume)
		}
		status, output, errMsg := finalizeStreamResult("goose", timeout, runCtx.Err(), writeErr, exitErr, emitted, streamTerminalState{
			lastAssistantText: state.lastAssistantText,
			finalResultText:   state.finalResultText,
			sawResult:         state.sawResult,
			resultIsError:     state.resultIsError,
			scanErr:           scanErr,
		}, "")
		if errMsg != "" {
			errMsg = withAgentStderr(errMsg, "goose", stderrBuf.Tail())
		}
		logStreamProtocolObservation(b.cfg.Logger, streamProtocolObservation{
			provider: "goose", cliVersion: b.cfg.CLIVersion, model: state.model,
			exitCode: streamProcessExitCode(exitErr), eventCount: state.eventCount,
			invalidEventCount: state.invalidEventCount, assistantEventCount: state.assistantEventCount,
			toolUseCount: state.toolUseCount, sawResult: state.sawResult, resultIsError: state.resultIsError,
			resultBytes: len(state.finalResultText), lastAssistantBytes: len(state.lastAssistantText),
			scannerError: scanErr != nil, lastEventType: state.lastEventType,
			unreadableAssistantCount: state.unreadableAssistantCount,
		})
		b.cfg.Logger.Info("goose finished", "pid", cmd.Process.Pid, "status", status, "duration", duration.Round(time.Millisecond).String())
		resCh <- Result{
			Status: status, Output: output, Error: errMsg, DurationMs: duration.Milliseconds(),
			SessionID:      resolveSessionID(string(resume), emitted, status == "failed", errMsg),
			Usage:          state.usage,
			ResumeRejected: resumeWasRejected(string(resume), string(state.sessionID), status == "failed", errMsg),
		}
	}()
	return &Session{Messages: msgCh, Result: resCh}, nil
}

// gooseStreamEvent is the 1.48.0 StreamEvent wire shape
// (tag = "type", rename_all = "snake_case"): message, notification, error, complete.
// Session id is not on that enum. The optional session_id field is accepted
// only when it parses, so a later Goose release can start emitting one.
type gooseStreamEvent struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id,omitempty"`
	Message   json.RawMessage `json:"message,omitempty"`
	Error     string          `json:"error,omitempty"`
	gooseCompleteUsage
}

type gooseCompleteUsage struct {
	TotalTokens           *int64 `json:"total_tokens,omitempty"`
	InputTokens           *int64 `json:"input_tokens,omitempty"`
	OutputTokens          *int64 `json:"output_tokens,omitempty"`
	CacheReadInputTokens  *int64 `json:"cache_read_input_tokens,omitempty"`
	CacheWriteInputTokens *int64 `json:"cache_write_input_tokens,omitempty"`
}

type gooseMessage struct {
	Role     string              `json:"role"`
	Content  []gooseContentBlock `json:"content"`
	Metadata *gooseMetadata      `json:"metadata,omitempty"`
}

type gooseMetadata struct {
	Inference *gooseInference `json:"inference,omitempty"`
	Usage     *gooseUsage     `json:"usage,omitempty"`
}

type gooseInference struct {
	Provider       string `json:"provider"`
	RequestedModel string `json:"requestedModel"`
	ResolvedModel  string `json:"resolvedModel"`
}

type gooseUsage struct {
	InputTokens      int64 `json:"inputTokens"`
	OutputTokens     int64 `json:"outputTokens"`
	CacheReadTokens  int64 `json:"cacheReadTokens"`
	CacheWriteTokens int64 `json:"cacheWriteTokens"`
}

type gooseContentBlock struct {
	Type       string          `json:"type"`
	Text       string          `json:"text,omitempty"`
	Thinking   string          `json:"thinking,omitempty"`
	ID         string          `json:"id,omitempty"`
	ToolCall   json.RawMessage `json:"toolCall,omitempty"`
	ToolResult json.RawMessage `json:"toolResult,omitempty"`
}

type gooseStreamState struct {
	sessionID                                                        gooseSessionID
	model, lastAssistantText, finalResultText, lastEventType         string
	sawResult, resultIsError                                         bool
	usage                                                            map[string]TokenUsage
	eventCount, invalidEventCount, assistantEventCount, toolUseCount int
	unreadableAssistantCount                                         int
}

func handleGooseEvent(event gooseStreamEvent, ch chan<- Message, state *gooseStreamState) {
	if id, ok := parseGooseSessionID(event.SessionID); ok {
		state.sessionID = id
	}
	switch event.Type {
	case "message":
		turn, model := handleGooseMessage(event.Message, ch, state.usage)
		if model != "" {
			state.model = model
		}
		if turn.toolUses > 0 || turn.text != "" || turn.understood {
			if turn.text != "" || turn.toolUses > 0 {
				state.assistantEventCount++
			}
		}
		state.toolUseCount += turn.toolUses
		if !turn.understood {
			state.unreadableAssistantCount++
		}
		state.lastAssistantText = turn.resolveFallback(state.lastAssistantText)
	case "notification":
		trySend(ch, Message{Type: MessageStatus, Status: "running", SessionID: string(state.sessionID)})
	case "error":
		state.sawResult = true
		state.resultIsError = true
		if event.Error != "" {
			state.finalResultText = event.Error
		} else {
			state.finalResultText = "goose returned an error event without details"
		}
	case "complete":
		state.sawResult = true
		state.resultIsError = false
		if usage := gooseCompleteTokenUsage(event.gooseCompleteUsage, state.model); len(usage) > 0 {
			state.usage = usage
		}
	}
}

func handleGooseMessage(raw json.RawMessage, ch chan<- Message, usage map[string]TokenUsage) (assistantTurn, string) {
	var message gooseMessage
	if json.Unmarshal(raw, &message) != nil {
		return assistantTurn{}, ""
	}
	model := gooseMessageModel(message)
	if message.Metadata != nil && message.Metadata.Usage != nil && model != "" {
		usage[model] = TokenUsage{
			InputTokens:      message.Metadata.Usage.InputTokens,
			OutputTokens:     message.Metadata.Usage.OutputTokens,
			CacheReadTokens:  message.Metadata.Usage.CacheReadTokens,
			CacheWriteTokens: message.Metadata.Usage.CacheWriteTokens,
		}
	}
	if message.Role == "user" {
		handleGooseUser(message, ch)
		return assistantTurn{understood: true}, model
	}
	turn := assistantTurn{understood: true}
	var text strings.Builder
	tools := 0
	for _, block := range message.Content {
		switch block.Type {
		case "thinking":
			if block.Thinking != "" {
				trySend(ch, Message{Type: MessageThinking, Content: block.Thinking})
			}
		case "redactedThinking":
		case "text":
			if block.Text != "" {
				text.WriteString(block.Text)
				trySend(ch, Message{Type: MessageText, Content: block.Text})
			}
		case "toolRequest":
			tools++
			name, input, callID := gooseToolRequest(block)
			trySend(ch, Message{Type: MessageToolUse, Tool: name, CallID: callID, Input: input})
		case "image", "toolConfirmationRequest", "actionRequired", "frontendToolRequest", "systemNotification":
		default:
			turn.understood = false
		}
	}
	turn.text = text.String()
	turn.toolUses = tools
	return turn, model
}

func handleGooseUser(message gooseMessage, ch chan<- Message) {
	for _, block := range message.Content {
		if block.Type == "toolResponse" {
			trySend(ch, Message{Type: MessageToolResult, CallID: block.ID, Output: gooseToolResultOutput(block.ToolResult)})
		}
	}
}

func gooseMessageModel(message gooseMessage) string {
	if message.Metadata == nil || message.Metadata.Inference == nil {
		return ""
	}
	if model := strings.TrimSpace(message.Metadata.Inference.ResolvedModel); model != "" {
		return model
	}
	return strings.TrimSpace(message.Metadata.Inference.RequestedModel)
}

func gooseToolRequest(block gooseContentBlock) (string, map[string]any, string) {
	callID := block.ID
	name, input := gooseToolCall(block.ToolCall)
	return name, input, callID
}

func gooseToolCall(raw json.RawMessage) (string, map[string]any) {
	if len(raw) == 0 {
		return "", nil
	}
	var direct struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if json.Unmarshal(raw, &direct) == nil && direct.Name != "" {
		return direct.Name, gooseArgsMap(direct.Arguments)
	}
	var wrapped struct {
		Ok *struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"Ok"`
	}
	if json.Unmarshal(raw, &wrapped) == nil && wrapped.Ok != nil && wrapped.Ok.Name != "" {
		return wrapped.Ok.Name, gooseArgsMap(wrapped.Ok.Arguments)
	}
	return "", nil
}

func gooseArgsMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var input map[string]any
	if json.Unmarshal(raw, &input) != nil {
		return nil
	}
	return input
}

func gooseToolResultOutput(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var wrapped struct {
		Ok json.RawMessage `json:"Ok"`
	}
	if json.Unmarshal(raw, &wrapped) == nil && len(wrapped.Ok) > 0 {
		return gooseToolResultOutput(wrapped.Ok)
	}
	return string(raw)
}

func gooseCompleteTokenUsage(usage gooseCompleteUsage, model string) map[string]TokenUsage {
	var in, out, cache int64
	if usage.InputTokens != nil {
		in = *usage.InputTokens
	}
	if usage.OutputTokens != nil {
		out = *usage.OutputTokens
	}
	if usage.CacheReadInputTokens != nil {
		cache = *usage.CacheReadInputTokens
	}
	if in == 0 && out == 0 && cache == 0 {
		return nil
	}
	if model == "" {
		model = "goose"
	}
	return map[string]TokenUsage{model: {InputTokens: in, OutputTokens: out, CacheReadTokens: cache}}
}
