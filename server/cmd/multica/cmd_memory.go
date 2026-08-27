package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/multica-ai/multica/server/internal/cli"
)

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Pin, list, search, and forget Multica memory notes",
}

var memoryAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Pin a note on a bank",
	RunE:  runMemoryAdd,
}

var memoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List notes in a bank",
	RunE:  runMemoryList,
}

var memorySearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search notes in a bank",
	RunE:  runMemorySearch,
}

var memoryGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get one note",
	Args:  exactArgs(1),
	RunE:  runMemoryGet,
}

var memoryRecallCmd = &cobra.Command{
	Use:   "recall",
	Short: "Recall notes across the current ancestry",
	RunE:  runMemoryRecall,
}

var memoryForgetCmd = &cobra.Command{
	Use:   "forget <id>",
	Short: "Forget a note",
	Args:  exactArgs(1),
	RunE:  runMemoryForget,
}

func init() {
	memoryCmd.AddCommand(memoryAddCmd)
	memoryCmd.AddCommand(memoryListCmd)
	memoryCmd.AddCommand(memorySearchCmd)
	memoryCmd.AddCommand(memoryGetCmd)
	memoryCmd.AddCommand(memoryRecallCmd)
	memoryCmd.AddCommand(memoryForgetCmd)

	memoryAddCmd.Flags().String("scope", "", "Bank scope: workspace, initiative, project, issue, squad, agent, user (required)")
	memoryAddCmd.Flags().String("owner-id", "", "Owner UUID for that scope (required)")
	memoryAddCmd.Flags().String("body", "", "Note body (required)")
	memoryAddCmd.Flags().String("kind", "fact", "Kind: fact, preference, procedure, observation")
	memoryAddCmd.Flags().String("output", "json", "Output format: table or json")

	memoryListCmd.Flags().String("scope", "", "Bank scope (required)")
	memoryListCmd.Flags().String("owner-id", "", "Owner UUID (required)")
	memoryListCmd.Flags().String("q", "", "Optional body search")
	memoryListCmd.Flags().Int("limit", 50, "Max rows")
	memoryListCmd.Flags().String("output", "table", "Output format: table or json")
	memoryListCmd.Flags().Bool("full-id", false, "Show full UUIDs in table output")

	memorySearchCmd.Flags().String("scope", "", "Bank scope (required)")
	memorySearchCmd.Flags().String("owner-id", "", "Owner UUID (required)")
	memorySearchCmd.Flags().String("q", "", "Search query (required)")
	memorySearchCmd.Flags().Int("limit", 50, "Max rows")
	memorySearchCmd.Flags().String("output", "table", "Output format: table or json")
	memorySearchCmd.Flags().Bool("full-id", false, "Show full UUIDs in table output")

	memoryGetCmd.Flags().String("output", "json", "Output format: table or json")

	memoryRecallCmd.Flags().String("q", "", "Optional search query")
	memoryRecallCmd.Flags().String("issue-id", "", "Include this issue bank")
	memoryRecallCmd.Flags().String("project-id", "", "Include this project bank")
	memoryRecallCmd.Flags().String("initiative-id", "", "Include this initiative bank")
	memoryRecallCmd.Flags().String("squad-id", "", "Include this squad bank")
	memoryRecallCmd.Flags().String("agent-id", "", "Include this agent bank")
	memoryRecallCmd.Flags().String("output", "table", "Output format: table or json")
	memoryRecallCmd.Flags().Bool("full-id", false, "Show full UUIDs in table output")

	memoryForgetCmd.Flags().String("output", "json", "Output format: table or json")
}

func runMemoryAdd(cmd *cobra.Command, _ []string) error {
	scope, _ := cmd.Flags().GetString("scope")
	ownerID, _ := cmd.Flags().GetString("owner-id")
	body, _ := cmd.Flags().GetString("body")
	kind, _ := cmd.Flags().GetString("kind")
	if strings.TrimSpace(scope) == "" || strings.TrimSpace(ownerID) == "" || strings.TrimSpace(body) == "" {
		return fmt.Errorf("--scope, --owner-id, and --body are required")
	}
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()
	var result map[string]any
	if err := client.PostJSON(ctx, "/api/memory", map[string]any{
		"scope":    scope,
		"owner_id": ownerID,
		"body":     body,
		"kind":     kind,
	}, &result); err != nil {
		return fmt.Errorf("add memory: %w", err)
	}
	return printMemoryEntry(cmd, result)
}

func runMemoryList(cmd *cobra.Command, _ []string) error {
	return runMemoryBankList(cmd, false)
}

func runMemorySearch(cmd *cobra.Command, _ []string) error {
	q, _ := cmd.Flags().GetString("q")
	if strings.TrimSpace(q) == "" {
		return fmt.Errorf("--q is required")
	}
	return runMemoryBankList(cmd, true)
}

func runMemoryBankList(cmd *cobra.Command, requireQuery bool) error {
	scope, _ := cmd.Flags().GetString("scope")
	ownerID, _ := cmd.Flags().GetString("owner-id")
	if strings.TrimSpace(scope) == "" || strings.TrimSpace(ownerID) == "" {
		return fmt.Errorf("--scope and --owner-id are required")
	}
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()
	params := url.Values{}
	params.Set("scope", scope)
	params.Set("owner_id", ownerID)
	if q, _ := cmd.Flags().GetString("q"); strings.TrimSpace(q) != "" {
		params.Set("q", q)
	} else if requireQuery {
		return fmt.Errorf("--q is required")
	}
	if limit, _ := cmd.Flags().GetInt("limit"); limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	var result map[string]any
	if err := client.GetJSON(ctx, "/api/memory?"+params.Encode(), &result); err != nil {
		return fmt.Errorf("list memory: %w", err)
	}
	return printMemoryList(cmd, result)
}

func runMemoryGet(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()
	var result map[string]any
	if err := client.GetJSON(ctx, "/api/memory/"+url.PathEscape(args[0]), &result); err != nil {
		return fmt.Errorf("get memory: %w", err)
	}
	return printMemoryEntry(cmd, result)
}

func runMemoryRecall(cmd *cobra.Command, _ []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()
	params := url.Values{}
	for _, key := range []string{"q", "issue-id", "project-id", "initiative-id", "squad-id", "agent-id"} {
		val, _ := cmd.Flags().GetString(key)
		if strings.TrimSpace(val) != "" {
			params.Set(strings.ReplaceAll(key, "-", "_"), val)
		}
	}
	path := "/api/memory/recall"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	var result map[string]any
	if err := client.GetJSON(ctx, path, &result); err != nil {
		return fmt.Errorf("recall memory: %w", err)
	}
	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, result)
	}
	hitsRaw, _ := result["hits"].([]any)
	fullID, _ := cmd.Flags().GetBool("full-id")
	headers := []string{"ID", "SCOPE", "KIND", "BODY"}
	rows := make([][]string, 0, len(hitsRaw))
	for _, raw := range hitsRaw {
		hit, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		rows = append(rows, []string{
			displayID(strVal(hit, "id"), fullID),
			strVal(hit, "scope"),
			strVal(hit, "kind"),
			strVal(hit, "body"),
		})
	}
	cli.PrintTable(os.Stdout, headers, rows)
	return nil
}

func runMemoryForget(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()
	if err := client.DeleteJSON(ctx, "/api/memory/"+url.PathEscape(args[0])); err != nil {
		return fmt.Errorf("forget memory: %w", err)
	}
	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, map[string]any{"id": args[0], "forgotten": true})
	}
	fmt.Fprintf(os.Stdout, "forgotten %s\n", args[0])
	return nil
}

func printMemoryEntry(cmd *cobra.Command, result map[string]any) error {
	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, result)
	}
	fmt.Fprintf(os.Stdout, "%s\t%s\t%s\n", strVal(result, "id"), strVal(result, "kind"), strVal(result, "body"))
	return nil
}

func printMemoryList(cmd *cobra.Command, result map[string]any) error {
	output, _ := cmd.Flags().GetString("output")
	entriesRaw, _ := result["entries"].([]any)
	if output == "json" {
		return cli.PrintJSON(os.Stdout, result)
	}
	fullID, _ := cmd.Flags().GetBool("full-id")
	headers := []string{"ID", "KIND", "BODY", "CREATED"}
	rows := make([][]string, 0, len(entriesRaw))
	for _, raw := range entriesRaw {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		created := strVal(entry, "created_at")
		if len(created) >= 10 {
			created = created[:10]
		}
		rows = append(rows, []string{
			displayID(strVal(entry, "id"), fullID),
			strVal(entry, "kind"),
			strVal(entry, "body"),
			created,
		})
	}
	cli.PrintTable(os.Stdout, headers, rows)
	return nil
}
