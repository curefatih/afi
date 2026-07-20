package platform

import "testing"

func TestSanitizeReturnTo(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"/app/dashboard", "/app/dashboard"},
		{"  /app/dashboard  ", "/app/dashboard"},
		{"https://evil.example/", ""},
		{"//evil.example", ""},
		{`/\evil.example`, ""},
		{`/app/\..\evil`, ""},
		{"app/dashboard", ""},
	}
	for _, tc := range cases {
		if got := sanitizeReturnTo(tc.in); got != tc.want {
			t.Fatalf("sanitizeReturnTo(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
