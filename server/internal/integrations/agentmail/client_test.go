package agentmail

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewUsesLiveClient(t *testing.T) {
	svc, err := New(Config{Box: testBox(t)}, testQ)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, ok := svc.api.(liveClient); !ok {
		t.Fatalf("api = %T, want liveClient", svc.api)
	}
}

func TestLiveClientEnsurePod(t *testing.T) {
	var gotAuth, gotClientID string
	client := newTestLive(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/pods" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		var body map[string]string
		readJSON(t, r, &body)
		gotClientID = body["client_id"]
		writeJSON(t, w, http.StatusOK, map[string]string{"pod_id": "pod_1", "client_id": body["client_id"]})
	})

	id, err := client.ensurePod(context.Background(), "am_org", "ws-1")
	if err != nil {
		t.Fatalf("ensurePod: %v", err)
	}
	if id != "pod_1" {
		t.Fatalf("pod id = %q", id)
	}
	if gotAuth != "Bearer am_org" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotClientID != "ws-1" {
		t.Fatalf("client_id = %q", gotClientID)
	}
}

func TestLiveClientEnsurePodConflictBody(t *testing.T) {
	client := newTestLive(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusConflict, map[string]string{"pod_id": "pod_existing", "client_id": "ws-1"})
	})

	id, err := client.ensurePod(context.Background(), "am_org", "ws-1")
	if err != nil {
		t.Fatalf("ensurePod: %v", err)
	}
	if id != "pod_existing" {
		t.Fatalf("pod id = %q", id)
	}
}

func TestLiveClientEnsureInboxUsesPodPath(t *testing.T) {
	var path string
	var body map[string]string
	client := newTestLive(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		readJSON(t, r, &body)
		writeJSON(t, w, http.StatusOK, map[string]string{
			"inbox_id": "inb_1",
			"email":    "ada@acme.test",
		})
	})

	inbox, err := client.ensureInbox(context.Background(), clientCred{apiKey: "am_org", podID: "pod_1"}, "agent-1", "Ada", InboxAddress{
		Username: "ada",
		Domain:   "acme.test",
	})
	if err != nil {
		t.Fatalf("ensureInbox: %v", err)
	}
	if inbox.id != "inb_1" || inbox.address != "ada@acme.test" {
		t.Fatalf("inbox = %+v", inbox)
	}
	if path != "/pods/pod_1/inboxes" {
		t.Fatalf("path = %q", path)
	}
	if body["username"] != "ada" || body["domain"] != "acme.test" {
		t.Fatalf("body = %+v", body)
	}
}

func TestLiveClientEnsureInboxOrgPath(t *testing.T) {
	var path string
	client := newTestLive(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		writeJSON(t, w, http.StatusOK, map[string]string{
			"inbox_id": "inb_2",
			"email":    "ada@custom.test",
		})
	})

	inbox, err := client.ensureInbox(context.Background(), clientCred{apiKey: "am_byo"}, "agent-1", "Ada", InboxAddress{})
	if err != nil {
		t.Fatalf("ensureInbox: %v", err)
	}
	if inbox.address != "ada@custom.test" {
		t.Fatalf("address = %q", inbox.address)
	}
	if path != "/inboxes" {
		t.Fatalf("path = %q", path)
	}
}

func TestLiveClientCreateInboxKeyPermissions(t *testing.T) {
	var perms map[string]bool
	var path string
	client := newTestLive(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		var body struct {
			Permissions map[string]bool `json:"permissions"`
		}
		readJSON(t, r, &body)
		perms = body.Permissions
		writeJSON(t, w, http.StatusOK, map[string]string{"api_key": "am_inbox_1"})
	})

	key, err := client.createInboxKey(context.Background(), clientCred{apiKey: "am_org"}, "user@agentmail.to")
	if err != nil {
		t.Fatalf("createInboxKey: %v", err)
	}
	if key != "am_inbox_1" {
		t.Fatalf("key = %q", key)
	}
	if path != "/inboxes/user@agentmail.to/api-keys" && path != "/inboxes/user%40agentmail.to/api-keys" {
		t.Fatalf("path = %q", path)
	}
	want := []string{"inbox_read", "message_read", "message_send", "draft_read", "draft_create", "draft_send"}
	for _, name := range want {
		if !perms[name] {
			t.Fatalf("missing permission %s: %+v", name, perms)
		}
	}
	for _, name := range []string{"inbox_create", "inbox_delete", "api_key_create", "pod_create", "domain_create"} {
		if perms[name] {
			t.Fatalf("management permission %s must stay off", name)
		}
	}
}

func TestLiveClientDeleteInboxUsesPodPath(t *testing.T) {
	var paths []string
	client := newTestLive(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/inboxes/") && strings.HasSuffix(r.URL.Path, "/api-keys"):
			writeJSON(t, w, http.StatusOK, map[string]any{
				"api_keys": []map[string]string{{"api_key_id": "key_1"}},
			})
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/api-keys/"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/inboxes/"):
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	if err := client.deleteInbox(context.Background(), clientCred{apiKey: "am_org", podID: "pod_1"}, "ada@agentmail.to"); err != nil {
		t.Fatalf("deleteInbox: %v", err)
	}
	joined := strings.Join(paths, "\n")
	if !strings.Contains(joined, "GET /inboxes/") || !strings.Contains(joined, "/api-keys") {
		t.Fatalf("did not list inbox keys:\n%s", joined)
	}
	if !strings.Contains(joined, "DELETE /inboxes/") || !strings.Contains(joined, "/api-keys/key_1") {
		t.Fatalf("did not delete inbox keys:\n%s", joined)
	}
	if !strings.Contains(joined, "DELETE /pods/pod_1/inboxes/ada@agentmail.to") &&
		!strings.Contains(joined, "DELETE /pods/pod_1/inboxes/ada%40agentmail.to") {
		t.Fatalf("did not delete inbox on pod path:\n%s", joined)
	}
}

func TestLiveClientListAndGetInbox(t *testing.T) {
	client := newTestLive(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/pods/pod_1/inboxes":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"inboxes": []map[string]string{{
					"inbox_id":     "inb_1",
					"email":        "ada@agentmail.to",
					"display_name": "Ada",
				}},
			})
		case r.Method == http.MethodGet && (r.URL.Path == "/pods/pod_1/inboxes/inb_1" || r.URL.Path == "/inboxes/inb_1"):
			writeJSON(t, w, http.StatusOK, map[string]any{
				"inbox_id": "inb_1",
				"email":    "ada@agentmail.to",
			})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	listed, err := client.listInboxes(context.Background(), clientCred{apiKey: "am_org", podID: "pod_1"})
	if err != nil {
		t.Fatalf("listInboxes: %v", err)
	}
	if len(listed) != 1 || listed[0].id != "inb_1" || listed[0].address != "ada@agentmail.to" {
		t.Fatalf("listed = %+v", listed)
	}
	got, err := client.getInbox(context.Background(), clientCred{apiKey: "am_org", podID: "pod_1"}, "inb_1")
	if err != nil {
		t.Fatalf("getInbox: %v", err)
	}
	if got.id != "inb_1" || got.address != "ada@agentmail.to" {
		t.Fatalf("get = %+v", got)
	}
}

func TestLiveClientDeleteInboxNotFound(t *testing.T) {
	client := newTestLive(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	err := client.deleteInbox(context.Background(), clientCred{apiKey: "am_org"}, "inb_missing")
	if !errors.Is(err, errRemoteNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestLiveClientDeleteInboxEmpty(t *testing.T) {
	client := newTestLive(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("delete must not call remote with empty id")
	})
	err := client.deleteInbox(context.Background(), clientCred{apiKey: "am_org"}, "")
	if !errors.Is(err, errRemoteNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestLiveClientDeletePod(t *testing.T) {
	var path string
	client := newTestLive(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	if err := client.deletePod(context.Background(), "am_org", "pod_1"); err != nil {
		t.Fatalf("deletePod: %v", err)
	}
	if path != "/pods/pod_1" {
		t.Fatalf("path = %q", path)
	}
}

func TestLiveClientInspectKey(t *testing.T) {
	t.Run("organization", func(t *testing.T) {
		client := newTestLive(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/me":
				writeJSON(t, w, http.StatusOK, map[string]string{"scope_type": "organization", "scope_id": "org_1"})
			case "/domains":
				writeJSON(t, w, http.StatusOK, map[string]any{"domains": []map[string]string{{"domain": "acme.test"}}})
			default:
				t.Fatalf("unexpected %s", r.URL.Path)
			}
		})
		info, err := client.inspectKey(context.Background(), "am_org")
		if err != nil {
			t.Fatalf("inspectKey: %v", err)
		}
		if info.authorityKind != authorityBYOOrg || info.podID != "" || info.domain != "acme.test" {
			t.Fatalf("info = %+v", info)
		}
	})
	t.Run("pod", func(t *testing.T) {
		client := newTestLive(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/me":
				writeJSON(t, w, http.StatusOK, map[string]string{"scope_type": "pod", "pod_id": "pod_9"})
			case "/domains":
				writeJSON(t, w, http.StatusOK, map[string]any{"domains": []any{}})
			default:
				t.Fatalf("unexpected %s", r.URL.Path)
			}
		})
		info, err := client.inspectKey(context.Background(), "am_pod")
		if err != nil {
			t.Fatalf("inspectKey: %v", err)
		}
		if info.authorityKind != authorityBYOPod || info.podID != "pod_9" {
			t.Fatalf("info = %+v", info)
		}
	})
	t.Run("inbox", func(t *testing.T) {
		client := newTestLive(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]string{"scope_type": "inbox", "inbox_id": "inb_1"})
		})
		_, err := client.inspectKey(context.Background(), "am_inbox")
		if !errors.Is(err, ErrBadOrgKey) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("unauthorized", func(t *testing.T) {
		client := newTestLive(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
		_, err := client.inspectKey(context.Background(), "am_bad")
		if !errors.Is(err, ErrBadOrgKey) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestLiveClientListAndGetThreads(t *testing.T) {
	client := newTestLive(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer am_inbox" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/inboxes/inb_1/threads":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"threads": []map[string]any{{
					"thread_id":     "thr_1",
					"subject":       "Hello",
					"preview":       "plain preview",
					"senders":       []string{"pat@example.com"},
					"recipients":    []string{"ada@agentmail.to"},
					"timestamp":     "2024-01-15T09:30:00Z",
					"message_count": 1,
				}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/inboxes/inb_1/threads/thr_1":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"thread_id": "thr_1",
				"subject":   "Hello",
				"messages": []map[string]any{{
					"message_id":     "msg_1",
					"from":           "pat@example.com",
					"to":             []string{"ada@agentmail.to"},
					"text":           "",
					"extracted_text": "from extract",
					"preview":        "from preview",
					"html":           "<script>alert(1)</script>",
				}},
			})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})

	page, err := client.listThreads(context.Background(), "am_inbox", "inb_1", threadQuery{})
	if err != nil {
		t.Fatalf("listThreads: %v", err)
	}
	if len(page.Threads) != 1 || page.Threads[0].ID != "thr_1" {
		t.Fatalf("page = %+v", page)
	}

	detail, err := client.getThread(context.Background(), "am_inbox", "inb_1", "thr_1")
	if err != nil {
		t.Fatalf("getThread: %v", err)
	}
	if len(detail.Messages) != 1 || detail.Messages[0].Text != "from extract" {
		t.Fatalf("detail = %+v", detail)
	}
	raw, _ := json.Marshal(detail)
	if strings.Contains(string(raw), "<script>") || strings.Contains(string(raw), "html") {
		t.Fatalf("html leaked: %s", raw)
	}
}

func TestLiveClientRejectsEmptyKey(t *testing.T) {
	client := newTestLive(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("empty key must not hit the network")
	})
	if err := client.validateOrgKey(context.Background(), ""); !errors.Is(err, ErrBadOrgKey) {
		t.Fatalf("err = %v", err)
	}
}

func newTestLive(t *testing.T, h http.HandlerFunc) liveClient {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return newLiveClient(srv.Client(), srv.URL)
}

func readJSON(t *testing.T, r *http.Request, out any) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if err := json.Unmarshal(body, out); err != nil {
		t.Fatalf("unmarshal %q: %v", body, err)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("write json: %v", err)
	}
}

func TestLiveClientRemoteMessage(t *testing.T) {
	got := remoteMessage([]byte(`{"error":{"name":"not_found","message":"inbox gone"}}`))
	if !strings.Contains(got, "inbox gone") {
		t.Fatalf("message = %q", got)
	}
}
