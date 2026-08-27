package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
)

const testMemoryUUID = "22222222-2222-2222-2222-222222222222"

func newMemoryAddTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "add"}
	cmd.Flags().String("scope", "", "")
	cmd.Flags().String("owner-id", "", "")
	cmd.Flags().String("body", "", "")
	cmd.Flags().String("kind", "fact", "")
	cmd.Flags().String("output", "json", "")
	return cmd
}

func newMemoryListTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "list"}
	cmd.Flags().String("scope", "", "")
	cmd.Flags().String("owner-id", "", "")
	cmd.Flags().String("q", "", "")
	cmd.Flags().Int("limit", 50, "")
	cmd.Flags().String("output", "json", "")
	cmd.Flags().Bool("full-id", false, "")
	return cmd
}

func newMemoryForgetTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "forget"}
	cmd.Flags().String("output", "json", "")
	return cmd
}

func TestRunMemoryAddSendsExpectedRequest(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/memory" {
			t.Fatalf("path = %q, want /api/memory", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    testMemoryUUID,
			"scope": body["scope"],
			"body":  body["body"],
			"kind":  body["kind"],
		})
	}))
	defer srv.Close()
	setCLITestServerEnv(t, srv.URL)

	cmd := newMemoryAddTestCmd()
	_ = cmd.Flags().Set("scope", "issue")
	_ = cmd.Flags().Set("owner-id", testMemoryUUID)
	_ = cmd.Flags().Set("body", "prefer rebase")
	_ = cmd.Flags().Set("kind", "preference")

	out, err := captureStdout(t, func() error { return runMemoryAdd(cmd, nil) })
	if err != nil {
		t.Fatalf("runMemoryAdd: %v", err)
	}
	if body["scope"] != "issue" || body["body"] != "prefer rebase" || body["kind"] != "preference" {
		t.Fatalf("body = %#v", body)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if got["id"] != testMemoryUUID {
		t.Fatalf("output id = %v", got["id"])
	}
}

func TestRunMemoryListSendsQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/memory" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("scope") != "workspace" {
			t.Fatalf("scope = %q", r.URL.Query().Get("scope"))
		}
		if r.URL.Query().Get("q") != "Thursday" {
			t.Fatalf("q = %q", r.URL.Query().Get("q"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"entries": []any{}, "total": 0})
	}))
	defer srv.Close()
	setCLITestServerEnv(t, srv.URL)

	cmd := newMemoryListTestCmd()
	_ = cmd.Flags().Set("scope", "workspace")
	_ = cmd.Flags().Set("owner-id", testMemoryUUID)
	_ = cmd.Flags().Set("q", "Thursday")
	if err := runMemoryList(cmd, nil); err != nil {
		t.Fatalf("runMemoryList: %v", err)
	}
}

func TestRunMemoryForgetSendsDelete(t *testing.T) {
	var sawDelete bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/api/memory/"+testMemoryUUID {
			t.Fatalf("path = %q", r.URL.Path)
		}
		sawDelete = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	setCLITestServerEnv(t, srv.URL)

	cmd := newMemoryForgetTestCmd()
	if err := runMemoryForget(cmd, []string{testMemoryUUID}); err != nil {
		t.Fatalf("runMemoryForget: %v", err)
	}
	if !sawDelete {
		t.Fatal("expected DELETE")
	}
}
