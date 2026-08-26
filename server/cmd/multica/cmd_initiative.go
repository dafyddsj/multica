package main

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"github.com/multica-ai/multica/server/internal/cli"
)

var initiativeCmd = &cobra.Command{
	Use:   "initiative",
	Short: "Work with initiatives",
}

var initiativeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List initiatives in the workspace",
	RunE:  runInitiativeList,
}

var initiativeGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get initiative details",
	Args:  exactArgs(1),
	RunE:  runInitiativeGet,
}

var initiativeCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new initiative",
	RunE:  runInitiativeCreate,
}

var initiativeUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an initiative",
	Args:  exactArgs(1),
	RunE:  runInitiativeUpdate,
}

var initiativeDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an initiative (detaches child projects)",
	Args:  exactArgs(1),
	RunE:  runInitiativeDelete,
}

var initiativeStatusCmd = &cobra.Command{
	Use:   "status <id> <status>",
	Short: "Change initiative status",
	Args:  exactArgs(2),
	RunE:  runInitiativeStatus,
}

func init() {
	initiativeCmd.AddCommand(initiativeListCmd)
	initiativeCmd.AddCommand(initiativeGetCmd)
	initiativeCmd.AddCommand(initiativeCreateCmd)
	initiativeCmd.AddCommand(initiativeUpdateCmd)
	initiativeCmd.AddCommand(initiativeDeleteCmd)
	initiativeCmd.AddCommand(initiativeStatusCmd)

	initiativeListCmd.Flags().String("output", "table", "Output format: table or json")
	initiativeListCmd.Flags().Bool("full-id", false, "Show full UUIDs in table output")
	initiativeListCmd.Flags().String("status", "", "Filter by status")

	initiativeGetCmd.Flags().String("output", "json", "Output format: table or json")

	initiativeCreateCmd.Flags().String("title", "", "Initiative title (required)")
	initiativeCreateCmd.Flags().String("description", "", "Initiative description")
	initiativeCreateCmd.Flags().String("status", "", "Initiative status")
	initiativeCreateCmd.Flags().String("icon", "", "Initiative icon (emoji)")
	initiativeCreateCmd.Flags().String("lead", "", "Lead name (member or agent)")
	initiativeCreateCmd.Flags().String("start-date", "", "Start date (calendar day, YYYY-MM-DD)")
	initiativeCreateCmd.Flags().String("due-date", "", "Due date (calendar day, YYYY-MM-DD)")
	initiativeCreateCmd.Flags().String("output", "json", "Output format: table or json")

	initiativeUpdateCmd.Flags().String("title", "", "New title")
	initiativeUpdateCmd.Flags().String("description", "", "New description")
	initiativeUpdateCmd.Flags().String("status", "", "New status")
	initiativeUpdateCmd.Flags().String("icon", "", "New icon (emoji)")
	initiativeUpdateCmd.Flags().String("lead", "", "New lead name (member or agent)")
	initiativeUpdateCmd.Flags().String("start-date", "", "New start date (calendar day, YYYY-MM-DD; pass empty string to clear)")
	initiativeUpdateCmd.Flags().String("due-date", "", "New due date (calendar day, YYYY-MM-DD; pass empty string to clear)")
	initiativeUpdateCmd.Flags().String("output", "json", "Output format: table or json")

	initiativeDeleteCmd.Flags().String("output", "json", "Output format: table or json")

	initiativeStatusCmd.Flags().String("output", "table", "Output format: table or json")
}

func runInitiativeList(cmd *cobra.Command, _ []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	params := url.Values{}
	if client.WorkspaceID != "" {
		params.Set("workspace_id", client.WorkspaceID)
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		params.Set("status", v)
	}

	path := "/api/initiatives"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var result map[string]any
	if err := client.GetJSON(ctx, path, &result); err != nil {
		return fmt.Errorf("list initiatives: %w", err)
	}

	initiativesRaw, _ := result["initiatives"].([]any)

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, initiativesRaw)
	}

	fullID, _ := cmd.Flags().GetBool("full-id")
	actors := loadActorDisplayLookup(ctx, client)
	headers := []string{"ID", "TITLE", "STATUS", "LEAD", "CREATED"}
	rows := make([][]string, 0, len(initiativesRaw))
	for _, raw := range initiativesRaw {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		lead := formatLead(item, actors)
		created := strVal(item, "created_at")
		if len(created) >= 10 {
			created = created[:10]
		}
		rows = append(rows, []string{
			displayID(strVal(item, "id"), fullID),
			strVal(item, "title"),
			strVal(item, "status"),
			lead,
			created,
		})
	}
	cli.PrintTable(os.Stdout, headers, rows)
	return nil
}

func runInitiativeGet(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	ref, err := resolveInitiativeID(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("resolve initiative: %w", err)
	}

	var initiative map[string]any
	if err := client.GetJSON(ctx, "/api/initiatives/"+ref.ID, &initiative); err != nil {
		return fmt.Errorf("get initiative: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "table" {
		actors := loadActorDisplayLookup(ctx, client)
		lead := formatLead(initiative, actors)
		headers := []string{"ID", "TITLE", "STATUS", "LEAD", "DESCRIPTION"}
		rows := [][]string{{
			strVal(initiative, "id"),
			strVal(initiative, "title"),
			strVal(initiative, "status"),
			lead,
			strVal(initiative, "description"),
		}}
		cli.PrintTable(os.Stdout, headers, rows)
		return nil
	}

	return cli.PrintJSON(os.Stdout, initiative)
}

func runInitiativeCreate(cmd *cobra.Command, _ []string) error {
	title, _ := cmd.Flags().GetString("title")
	if title == "" {
		return fmt.Errorf("--title is required")
	}

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	body := map[string]any{"title": title}
	if v, _ := cmd.Flags().GetString("description"); v != "" {
		body["description"] = v
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		if err := validateProjectStatus(v); err != nil {
			return err
		}
		body["status"] = v
	}
	if v, _ := cmd.Flags().GetString("icon"); v != "" {
		body["icon"] = v
	}
	if v, _ := cmd.Flags().GetString("lead"); v != "" {
		aType, aID, resolveErr := resolveAssignee(ctx, client, v, memberOrAgentKinds)
		if resolveErr != nil {
			return fmt.Errorf("resolve lead: %w", resolveErr)
		}
		body["lead_type"] = aType
		body["lead_id"] = aID
	}
	if v, _ := cmd.Flags().GetString("start-date"); v != "" {
		body["start_date"] = v
	}
	if v, _ := cmd.Flags().GetString("due-date"); v != "" {
		body["due_date"] = v
	}

	var result map[string]any
	if err := client.PostJSON(ctx, "/api/initiatives", body, &result); err != nil {
		return fmt.Errorf("create initiative: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "table" {
		headers := []string{"ID", "TITLE", "STATUS"}
		rows := [][]string{{
			strVal(result, "id"),
			strVal(result, "title"),
			strVal(result, "status"),
		}}
		cli.PrintTable(os.Stdout, headers, rows)
		return nil
	}

	return cli.PrintJSON(os.Stdout, result)
}

func runInitiativeUpdate(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	ref, err := resolveInitiativeID(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("resolve initiative: %w", err)
	}

	body := map[string]any{}
	if cmd.Flags().Changed("title") {
		v, _ := cmd.Flags().GetString("title")
		body["title"] = v
	}
	if cmd.Flags().Changed("description") {
		v, _ := cmd.Flags().GetString("description")
		body["description"] = v
	}
	if cmd.Flags().Changed("status") {
		v, _ := cmd.Flags().GetString("status")
		if err := validateProjectStatus(v); err != nil {
			return err
		}
		body["status"] = v
	}
	if cmd.Flags().Changed("icon") {
		v, _ := cmd.Flags().GetString("icon")
		body["icon"] = v
	}
	if cmd.Flags().Changed("lead") {
		v, _ := cmd.Flags().GetString("lead")
		aType, aID, resolveErr := resolveAssignee(ctx, client, v, memberOrAgentKinds)
		if resolveErr != nil {
			return fmt.Errorf("resolve lead: %w", resolveErr)
		}
		body["lead_type"] = aType
		body["lead_id"] = aID
	}
	if cmd.Flags().Changed("start-date") {
		v, _ := cmd.Flags().GetString("start-date")
		body["start_date"] = v
	}
	if cmd.Flags().Changed("due-date") {
		v, _ := cmd.Flags().GetString("due-date")
		body["due_date"] = v
	}

	if len(body) == 0 {
		return fmt.Errorf("no fields to update; use flags like --title, --status, --description, --icon, --lead, --start-date, --due-date")
	}

	var result map[string]any
	if err := client.PutJSON(ctx, "/api/initiatives/"+ref.ID, body, &result); err != nil {
		return fmt.Errorf("update initiative: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "table" {
		headers := []string{"ID", "TITLE", "STATUS"}
		rows := [][]string{{
			strVal(result, "id"),
			strVal(result, "title"),
			strVal(result, "status"),
		}}
		cli.PrintTable(os.Stdout, headers, rows)
		return nil
	}

	return cli.PrintJSON(os.Stdout, result)
}

func runInitiativeDelete(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	ref, err := resolveInitiativeID(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("resolve initiative: %w", err)
	}

	if err := client.DeleteJSON(ctx, "/api/initiatives/"+ref.ID); err != nil {
		return fmt.Errorf("delete initiative: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Initiative %s deleted. Child projects were detached, not deleted.\n", ref.Display)
	return nil
}

func runInitiativeStatus(cmd *cobra.Command, args []string) error {
	id := args[0]
	status := args[1]

	if err := validateProjectStatus(status); err != nil {
		return err
	}

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	ref, err := resolveInitiativeID(ctx, client, id)
	if err != nil {
		return fmt.Errorf("resolve initiative: %w", err)
	}

	body := map[string]any{"status": status}
	var result map[string]any
	if err := client.PutJSON(ctx, "/api/initiatives/"+ref.ID, body, &result); err != nil {
		return fmt.Errorf("update initiative status: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, result)
	}

	headers := []string{"ID", "TITLE", "STATUS"}
	rows := [][]string{{
		strVal(result, "id"),
		strVal(result, "title"),
		strVal(result, "status"),
	}}
	cli.PrintTable(os.Stdout, headers, rows)
	return nil
}
