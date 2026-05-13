package mcprpc

import "testing"

// TestCapitalizeASCIIScope pins the deprecated-strings.Title replacement
// helper. Drop 4c.5 D.2 swapped two `strings.Title(string(actionItem.Scope))`
// call sites in `instructionsExplain` to `capitalizeASCIIScope` to retire the
// Go 1.18 `strings.Title` deprecation. Inputs are the closed `KindAppliesTo`
// enum (pure ASCII), so the helper is a single-byte first-letter transform;
// the test pins both ASCII-lowercase capitalization and the no-op branches.
func TestCapitalizeASCIIScope(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty input returns empty", in: "", want: ""},
		{name: "all-lower ascii word capitalizes first letter", in: "build", want: "Build"},
		{name: "single lower letter capitalizes", in: "b", want: "B"},
		{name: "already capitalized passes through", in: "Build", want: "Build"},
		{name: "all-upper passes through", in: "BUILD", want: "BUILD"},
		{name: "leading digit passes through", in: "1build", want: "1build"},
		{name: "leading hyphen passes through", in: "-build", want: "-build"},
		{name: "mixed case keeps trailing letters", in: "buildQA", want: "BuildQA"},
		{name: "kind-appliesto droplet input", in: "droplet", want: "Droplet"},
		{name: "kind-appliesto plan input", in: "plan", want: "Plan"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := capitalizeASCIIScope(tc.in)
			if got != tc.want {
				t.Fatalf("capitalizeASCIIScope(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
