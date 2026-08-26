package handler

import "testing"

func TestParseCoAuthoredByEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "empty", in: "", want: ""},
		{name: "spaces", in: "   ", want: ""},
		{name: "plain", in: "Review@Example.com", want: "review@example.com"},
		{name: "display name", in: "Review Bot <review@example.com>", want: "review@example.com"},
		{name: "embedded newline", in: "review@exam\nple.com", wantErr: true},
		{name: "garbage", in: "not-an-email", wantErr: true},
		{name: "missing local", in: "@example.com", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseCoAuthoredByEmail(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseCoAuthoredByEmail(%q) = %q, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCoAuthoredByEmail(%q) error = %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("parseCoAuthoredByEmail(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
