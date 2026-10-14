package glob

import "testing"

func TestMatch(t *testing.T) {
	cases := []struct {
		pat, path string
		want      bool
	}{
		{"**/node_modules/**", "node_modules/pkg/index.js", true},
		{"**/node_modules/**", "web/node_modules/pkg/index.js", true},
		{"**/node_modules/**", "web/src/index.js", false},
		{"**/*_test.go", "internal/foo_test.go", true},
		{"**/*_test.go", "foo_test.go", true},
		{"**/*.test.*", "src/a.test.ts", true},
		{"**/*.test.*", "src/a.ts", false},
		{"**/*.spec.*", "src/a.spec.tsx", true},
		{"**/test_*.py", "tests/test_api.py", true},
		{"**/tests/**", "pkg/tests/a.go", true},
		{"*.go", "a.go", true},
		{"*.go", "pkg/a.go", false},
		{"**/generated/**", "src/generated/x.go", true},
	}
	for _, tc := range cases {
		if got := Match(tc.pat, tc.path); got != tc.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tc.pat, tc.path, got, tc.want)
		}
	}
}
