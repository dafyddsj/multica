package clerk

import "testing"

func TestFromEnvRequiresBothKeys(t *testing.T) {
	t.Setenv("CLERK_SECRET_KEY", "")
	t.Setenv("CLERK_PUBLISHABLE_KEY", "")
	if got := FromEnv(); got != nil {
		t.Fatalf("empty env: want nil, got %#v", got)
	}

	t.Setenv("CLERK_SECRET_KEY", "sk_test_x")
	t.Setenv("CLERK_PUBLISHABLE_KEY", "")
	if got := FromEnv(); got != nil {
		t.Fatalf("secret only: want nil client")
	}
	if PublishableKeyFromEnv() != "" {
		t.Fatalf("secret only: publishable key must stay hidden")
	}

	t.Setenv("CLERK_SECRET_KEY", "")
	t.Setenv("CLERK_PUBLISHABLE_KEY", "pk_test_x")
	if got := FromEnv(); got != nil {
		t.Fatalf("publishable only: want nil client")
	}
	if PublishableKeyFromEnv() != "" {
		t.Fatalf("publishable only: key must stay hidden until secret is set")
	}

	t.Setenv("CLERK_SECRET_KEY", "  sk_test_x  ")
	t.Setenv("CLERK_PUBLISHABLE_KEY", "  pk_test_x  ")
	got := FromEnv()
	if got == nil {
		t.Fatal("both keys: want a client")
	}
	if got.PublishableKey != "pk_test_x" {
		t.Fatalf("publishable key: got %q", got.PublishableKey)
	}
	if PublishableKeyFromEnv() != "pk_test_x" {
		t.Fatalf("PublishableKeyFromEnv: got %q", PublishableKeyFromEnv())
	}
}
