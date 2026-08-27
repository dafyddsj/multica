package memory

import "testing"

func TestWorkspaceEnabled(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "empty", raw: "", want: false},
		{name: "empty object", raw: "{}", want: false},
		{name: "false", raw: `{"memory_enabled": false}`, want: false},
		{name: "true", raw: `{"memory_enabled": true}`, want: true},
		{name: "string true", raw: `{"memory_enabled": "true"}`, want: false},
		{name: "invalid json", raw: `{`, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := WorkspaceEnabled([]byte(tc.raw)); got != tc.want {
				t.Fatalf("WorkspaceEnabled(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestValidateBody(t *testing.T) {
	t.Parallel()
	if ValidateBody("  ") != "body is required" {
		t.Fatal("empty body should be rejected")
	}
	if ValidateBody("ok") != "" {
		t.Fatal("short body should pass")
	}
}
