package agentmail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/multica-ai/multica/server/internal/util"
	"github.com/multica-ai/multica/server/internal/util/secretbox"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

const hostedOrgKey = "am_hosted_org_test_secret"

var (
	testPool *pgxpool.Pool
	testQ    *db.Queries
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://multica:multica@localhost:5432/multica?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Printf("agentmail tests: no database: %v\n", err)
		os.Exit(1)
	}
	if err := pool.Ping(ctx); err != nil {
		fmt.Printf("agentmail tests: database not reachable: %v\n", err)
		pool.Close()
		os.Exit(1)
	}
	if err := applyAgentMailMigrations(ctx, pool); err != nil {
		fmt.Printf("agentmail tests: apply migrations: %v\n", err)
		pool.Close()
		os.Exit(1)
	}
	testPool = pool
	testQ = db.New(pool)
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func applyAgentMailMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return errors.New("runtime.Caller failed")
	}
	dir := filepath.Join(filepath.Dir(file), "..", "..", "..", "migrations")
	for _, name := range []string{
		"459_agentmail_tables.up.sql",
		"460_agentmail_inbox_agent_index.up.sql",
		"461_agentmail_purge_workspace_index.up.sql",
	} {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		for _, stmt := range splitSQL(string(body)) {
			if _, err := pool.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	return nil
}

func splitSQL(src string) []string {
	var out []string
	var buf strings.Builder
	for _, line := range strings.Split(src, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "--") {
			continue
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
		if strings.HasSuffix(trim, ";") {
			out = append(out, buf.String())
			buf.Reset()
		}
	}
	if rest := strings.TrimSpace(buf.String()); rest != "" {
		out = append(out, rest)
	}
	return out
}

type fakeClient struct {
	pods       map[string]string
	inboxes    map[string]remoteInbox
	deleted    map[string]struct{}
	podKeys    map[string]string
	rejectKeys map[string]struct{}
	minted     []string
	ensurePodN atomic.Int32
	createKeyN atomic.Int32
	keySeq     atomic.Int32
}

func newFake() *fakeClient {
	return &fakeClient{
		pods:       map[string]string{},
		inboxes:    map[string]remoteInbox{},
		deleted:    map[string]struct{}{},
		podKeys:    map[string]string{},
		rejectKeys: map[string]struct{}{},
	}
}

func (f *fakeClient) ensurePod(_ context.Context, _, clientID string) (string, error) {
	f.ensurePodN.Add(1)
	if id, ok := f.pods[clientID]; ok {
		return id, nil
	}
	id := "pod_" + clientID
	f.pods[clientID] = id
	return id, nil
}

func (f *fakeClient) ensureInbox(_ context.Context, _ clientCred, clientID, _ string) (remoteInbox, error) {
	if inbox, ok := f.inboxes[clientID]; ok {
		if _, gone := f.deleted[inbox.id]; !gone {
			return inbox, nil
		}
	}
	id := "inb_" + uuid.NewString()
	inbox := remoteInbox{id: id, address: clientID[:8] + "@agentmail.to"}
	f.inboxes[clientID] = inbox
	delete(f.deleted, id)
	return inbox, nil
}

func (f *fakeClient) createInboxKey(_ context.Context, _ clientCred, _ string) (string, error) {
	f.createKeyN.Add(1)
	seq := f.keySeq.Add(1)
	key := fmt.Sprintf("am_inbox_%d_%s", seq, uuid.NewString()[:8])
	f.minted = append(f.minted, key)
	return key, nil
}

func (f *fakeClient) deleteInbox(_ context.Context, _ clientCred, inboxID string) error {
	if inboxID == "" {
		return errRemoteNotFound
	}
	found := false
	for _, inbox := range f.inboxes {
		if inbox.id == inboxID {
			found = true
			break
		}
	}
	if !found {
		if _, gone := f.deleted[inboxID]; gone {
			return errRemoteNotFound
		}
		return errRemoteNotFound
	}
	f.deleted[inboxID] = struct{}{}
	return nil
}

func (f *fakeClient) deletePod(_ context.Context, _, podID string) error {
	if podID == "" {
		return errRemoteNotFound
	}
	for clientID, id := range f.pods {
		if id == podID {
			delete(f.pods, clientID)
			return nil
		}
	}
	return errRemoteNotFound
}

func (f *fakeClient) validateOrgKey(_ context.Context, orgKey string) error {
	if orgKey == "" {
		return ErrBadOrgKey
	}
	if _, reject := f.rejectKeys[orgKey]; reject {
		return ErrBadOrgKey
	}
	return nil
}

func (f *fakeClient) inspectKey(ctx context.Context, orgKey string) (inspectedKey, error) {
	if err := f.validateOrgKey(ctx, orgKey); err != nil {
		return inspectedKey{}, err
	}
	if podID, ok := f.podKeys[orgKey]; ok {
		return inspectedKey{authorityKind: authorityBYOPod, podID: podID}, nil
	}
	return inspectedKey{authorityKind: authorityBYOOrg}, nil
}

func testBox(t *testing.T) *secretbox.Box {
	t.Helper()
	box, err := secretbox.New(make([]byte, secretbox.KeySize))
	if err != nil {
		t.Fatalf("secretbox.New: %v", err)
	}
	return box
}

func newTestService(t *testing.T, fake *fakeClient, cfg Config) *Service {
	t.Helper()
	if cfg.Box == nil {
		cfg.Box = testBox(t)
	}
	if cfg.HostedOrgKey == "" {
		cfg.HostedOrgKey = hostedOrgKey
	}
	svc, err := New(cfg, testQ)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	svc.api = fake
	return svc
}

func newIDs(t *testing.T) (wsID, actorID, agentID pgtype.UUID) {
	t.Helper()
	wsID = util.MustParseUUID(uuid.NewString())
	actorID = util.MustParseUUID(uuid.NewString())
	agentID = util.MustParseUUID(uuid.NewString())
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = testPool.Exec(ctx, `DELETE FROM agentmail_purge WHERE workspace_id = $1`, wsID)
		_, _ = testPool.Exec(ctx, `DELETE FROM agentmail_inbox WHERE workspace_id = $1`, wsID)
		_, _ = testPool.Exec(ctx, `DELETE FROM agentmail_connection WHERE workspace_id = $1`, wsID)
	})
	return wsID, actorID, agentID
}

func orgKeyEncrypted(t *testing.T, wsID pgtype.UUID) pgtype.Text {
	t.Helper()
	var enc pgtype.Text
	err := testPool.QueryRow(context.Background(),
		`SELECT org_key_encrypted FROM agentmail_connection WHERE workspace_id = $1`, wsID,
	).Scan(&enc)
	if err != nil {
		t.Fatalf("org_key_encrypted: %v", err)
	}
	return enc
}

func inboxState(t *testing.T, wsID, agentID pgtype.UUID) string {
	t.Helper()
	var state string
	err := testPool.QueryRow(context.Background(),
		`SELECT state FROM agentmail_inbox WHERE workspace_id = $1 AND agent_id = $2`, wsID, agentID,
	).Scan(&state)
	if err != nil {
		t.Fatalf("inbox state: %v", err)
	}
	return state
}

func TestServiceLifecycle(t *testing.T) {
	ctx := context.Background()

	t.Run("connect hosted twice is idempotent and stores no org key", func(t *testing.T) {
		fake := newFake()
		svc := newTestService(t, fake, Config{})
		wsID, actorID, _ := newIDs(t)

		first, err := svc.Connect(ctx, wsID, actorID, HostedCredential())
		if err != nil {
			t.Fatalf("first connect: %v", err)
		}
		if first.Source != sourceHosted || first.State != stateActive {
			t.Fatalf("first status = %+v", first)
		}
		second, err := svc.Connect(ctx, wsID, actorID, HostedCredential())
		if err != nil {
			t.Fatalf("second connect: %v", err)
		}
		if second != first {
			t.Fatalf("status drifted: first=%+v second=%+v", first, second)
		}
		if fake.ensurePodN.Load() == 0 {
			t.Fatal("ensurePod was not called")
		}
		enc := orgKeyEncrypted(t, wsID)
		if enc.Valid {
			t.Fatalf("hosted row stored org_key_encrypted = %q", enc.String)
		}
	})

	t.Run("connect BYO then hosted without disconnect is mode conflict", func(t *testing.T) {
		fake := newFake()
		svc := newTestService(t, fake, Config{})
		wsID, actorID, _ := newIDs(t)

		cred, err := ParseBYOCredential("am_byo_org_key")
		if err != nil {
			t.Fatalf("ParseBYOCredential: %v", err)
		}
		if _, err := svc.Connect(ctx, wsID, actorID, cred); err != nil {
			t.Fatalf("BYO connect: %v", err)
		}
		_, err = svc.Connect(ctx, wsID, actorID, HostedCredential())
		if !errors.Is(err, ErrModeConflict) {
			t.Fatalf("got %v, want ErrModeConflict", err)
		}
	})

	t.Run("grant without connection is not connected", func(t *testing.T) {
		fake := newFake()
		svc := newTestService(t, fake, Config{})
		wsID, actorID, agentID := newIDs(t)

		_, err := svc.GrantInbox(ctx, wsID, agentID, actorID, "Ada")
		if !errors.Is(err, ErrNotConnected) {
			t.Fatalf("got %v, want ErrNotConnected", err)
		}
	})

	t.Run("grant twice keeps one live minted key", func(t *testing.T) {
		fake := newFake()
		svc := newTestService(t, fake, Config{})
		wsID, actorID, agentID := newIDs(t)
		if _, err := svc.Connect(ctx, wsID, actorID, HostedCredential()); err != nil {
			t.Fatalf("connect: %v", err)
		}

		first, err := svc.GrantInbox(ctx, wsID, agentID, actorID, "Ada")
		if err != nil {
			t.Fatalf("first grant: %v", err)
		}
		second, err := svc.GrantInbox(ctx, wsID, agentID, actorID, "Ada")
		if err != nil {
			t.Fatalf("second grant: %v", err)
		}
		if first.Address == "" || first.Address != second.Address {
			t.Fatalf("addresses: first=%q second=%q", first.Address, second.Address)
		}
		if fake.createKeyN.Load() != 1 {
			t.Fatalf("createInboxKey calls = %d, want 1", fake.createKeyN.Load())
		}
	})

	t.Run("crash after mint recovers on retry without leaking lost or org keys", func(t *testing.T) {
		fake := newFake()
		svc := newTestService(t, fake, Config{})
		wsID, actorID, agentID := newIDs(t)
		if _, err := svc.Connect(ctx, wsID, actorID, HostedCredential()); err != nil {
			t.Fatalf("connect: %v", err)
		}

		svc.failPersistActiveOnce = true
		_, err := svc.GrantInbox(ctx, wsID, agentID, actorID, "Ada")
		if err == nil {
			t.Fatal("expected persist failure")
		}
		if inboxState(t, wsID, agentID) != stateMintingKey {
			t.Fatalf("state = %s, want minting_key", inboxState(t, wsID, agentID))
		}
		if len(fake.minted) != 1 {
			t.Fatalf("minted = %d, want 1", len(fake.minted))
		}
		lost := fake.minted[0]

		got, err := svc.GrantInbox(ctx, wsID, agentID, actorID, "Ada")
		if err != nil {
			t.Fatalf("retry grant: %v", err)
		}
		if got.State != stateActive {
			t.Fatalf("retry state = %s", got.State)
		}
		overlay, err := svc.ClaimOverlay(ctx, wsID, agentID)
		if err != nil {
			t.Fatalf("ClaimOverlay: %v", err)
		}
		body := string(overlay)
		if strings.Contains(body, lost) {
			t.Fatal("overlay contains the lost mint")
		}
		if strings.Contains(body, hostedOrgKey) {
			t.Fatal("overlay contains the hosted org key")
		}
	})

	t.Run("claim overlay carries inbox key only", func(t *testing.T) {
		fake := newFake()
		svc := newTestService(t, fake, Config{})
		wsID, actorID, agentID := newIDs(t)
		if _, err := svc.Connect(ctx, wsID, actorID, HostedCredential()); err != nil {
			t.Fatalf("connect: %v", err)
		}
		if _, err := svc.GrantInbox(ctx, wsID, agentID, actorID, "Ada"); err != nil {
			t.Fatalf("grant: %v", err)
		}

		overlay, err := svc.ClaimOverlay(ctx, wsID, agentID)
		if err != nil {
			t.Fatalf("ClaimOverlay: %v", err)
		}
		var parsed struct {
			MCPServers map[string]struct {
				Type    string            `json:"type"`
				URL     string            `json:"url"`
				Headers map[string]string `json:"headers"`
			} `json:"mcpServers"`
		}
		if err := json.Unmarshal(overlay, &parsed); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		entry, ok := parsed.MCPServers["agentmail"]
		if !ok {
			t.Fatalf("missing agentmail entry: %s", overlay)
		}
		key, ok := entry.Headers["x-api-key"]
		if !ok || key == "" {
			t.Fatalf("missing x-api-key: %+v", entry.Headers)
		}
		if len(fake.minted) != 1 || key != fake.minted[0] {
			t.Fatalf("header key %q, minted %v", key, fake.minted)
		}
		if strings.Contains(string(overlay), hostedOrgKey) {
			t.Fatal("overlay contains the hosted org key")
		}
	})

	t.Run("org ciphertext rejects inbox opener", func(t *testing.T) {
		box := testBox(t)
		sealed, err := sealOrgKey(box, "am_org_plain")
		if err != nil {
			t.Fatalf("sealOrgKey: %v", err)
		}
		if _, err := openInboxKey(box, sealed); err == nil {
			t.Fatal("opening org ciphertext with inbox opener succeeded")
		}
	})

	t.Run("revoke then claim overlay is nil", func(t *testing.T) {
		fake := newFake()
		svc := newTestService(t, fake, Config{})
		wsID, actorID, agentID := newIDs(t)
		if _, err := svc.Connect(ctx, wsID, actorID, HostedCredential()); err != nil {
			t.Fatalf("connect: %v", err)
		}
		if _, err := svc.GrantInbox(ctx, wsID, agentID, actorID, "Ada"); err != nil {
			t.Fatalf("grant: %v", err)
		}
		if err := svc.RevokeInbox(ctx, wsID, agentID); err != nil {
			t.Fatalf("revoke: %v", err)
		}
		overlay, err := svc.ClaimOverlay(ctx, wsID, agentID)
		if err != nil {
			t.Fatalf("ClaimOverlay: %v", err)
		}
		if overlay != nil {
			t.Fatalf("overlay = %s, want nil", overlay)
		}
	})

	t.Run("hosted quota rejects a second grant", func(t *testing.T) {
		fake := newFake()
		svc := newTestService(t, fake, Config{WorkspaceInboxLimit: 1})
		wsID, actorID, agentA := newIDs(t)
		agentB := util.MustParseUUID(uuid.NewString())
		if _, err := svc.Connect(ctx, wsID, actorID, HostedCredential()); err != nil {
			t.Fatalf("connect: %v", err)
		}
		if _, err := svc.GrantInbox(ctx, wsID, agentA, actorID, "Ada"); err != nil {
			t.Fatalf("first grant: %v", err)
		}
		_, err := svc.GrantInbox(ctx, wsID, agentB, actorID, "Bob")
		if !errors.Is(err, ErrInboxQuota) {
			t.Fatalf("got %v, want ErrInboxQuota", err)
		}
	})

	t.Run("sweep workspace removes connection and inbox rows", func(t *testing.T) {
		fake := newFake()
		svc := newTestService(t, fake, Config{})
		wsID, actorID, agentID := newIDs(t)
		if _, err := svc.Connect(ctx, wsID, actorID, HostedCredential()); err != nil {
			t.Fatalf("connect: %v", err)
		}
		if _, err := svc.GrantInbox(ctx, wsID, agentID, actorID, "Ada"); err != nil {
			t.Fatalf("grant: %v", err)
		}
		if err := svc.SweepWorkspace(ctx, testQ, wsID); err != nil {
			t.Fatalf("sweep: %v", err)
		}
		status, err := svc.GetWorkspace(ctx, wsID)
		if err != nil {
			t.Fatalf("GetWorkspace: %v", err)
		}
		if status != (WorkspaceStatus{}) {
			t.Fatalf("connection survived sweep: %+v", status)
		}
		inbox, err := svc.GetInbox(ctx, wsID, agentID)
		if err != nil {
			t.Fatalf("GetInbox: %v", err)
		}
		if inbox != nil {
			t.Fatalf("inbox survived sweep: %+v", inbox)
		}
	})

	t.Run("permission whitelist excludes management scopes", func(t *testing.T) {
		forbidden := []string{"inbox_create", "api_key_create", "pod_create"}
		for _, p := range inboxKeyPermissions {
			for _, bad := range forbidden {
				if p == bad {
					t.Fatalf("inboxKeyPermissions contains %q", bad)
				}
			}
			if strings.HasPrefix(p, "api_key_") || strings.HasPrefix(p, "pod_") || strings.HasPrefix(p, "domain_") {
				t.Fatalf("inboxKeyPermissions contains management scope %q", p)
			}
			if p == "inbox_delete" {
				t.Fatal("inboxKeyPermissions contains inbox_delete")
			}
		}
	})
}

func TestParseBYOCredentialRejectsEmpty(t *testing.T) {
	if _, err := ParseBYOCredential("  "); !errors.Is(err, ErrBadOrgKey) {
		t.Fatalf("got %v, want ErrBadOrgKey", err)
	}
}
