package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// dottedFixture builds a small project tree under fakeRepo for the resolver
// tests. The tree shape is:
//
//	project (slug=tillsyn-demo, id=proj-demo)
//	  ├── L1[0] = a-root          (id=a)
//	  ├── L1[1] = b-root           (id=b)
//	  │     ├── L2[0] = b-c0       (id=b-c0)
//	  │     └── L2[1] = b-c1       (id=b-c1)
//	  ├── L1[2] = c-root           (id=c)
//	  │     ├── L2[0] = c-c0       (id=c-c0)
//	  │     ├── L2[1] = c-c1       (id=c-c1)
//	  │     ├── L2[2] = c-c2       (id=c-c2)
//	  │     ├── L2[3] = c-c3       (id=c-c3)
//	  │     ├── L2[4] = c-c4       (id=c-c4)
//	  │     └── L2[5] = c-c5       (id=c-c5)
//	  │           └── L3[0] = c5-g (id=c-c5-g)
//	  └── L1[3] = d-tie            (id=d-tie-aaa) [same created_at as e]
//	       L1[4] = e-tie            (id=e-tie-zzz) [same created_at as d]
//
// d-tie and e-tie share a CreatedAt — id ASC tie-breaks deterministically
// (`d-tie-aaa` < `e-tie-zzz` lexicographically). created_at order is
// monotonically increasing in declaration order otherwise.
func dottedFixture(t *testing.T) (*fakeRepo, string) {
	t.Helper()

	ctx := context.Background()
	repo := newFakeRepo()

	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("proj-demo", "Tillsyn Demo", "fixture", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	column, err := domain.NewColumn("col-todo", project.ID, "Todo", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	type spec struct {
		id        string
		parentID  string
		title     string
		createdAt time.Time
	}
	specs := []spec{
		{id: "a", parentID: "", title: "A root", createdAt: now.Add(1 * time.Second)},
		{id: "b", parentID: "", title: "B root", createdAt: now.Add(2 * time.Second)},
		{id: "c", parentID: "", title: "C root", createdAt: now.Add(3 * time.Second)},
		// d-tie and e-tie share createdAt; id ASC tie-breaks (d < e).
		{id: "d-tie-aaa", parentID: "", title: "D tie", createdAt: now.Add(4 * time.Second)},
		{id: "e-tie-zzz", parentID: "", title: "E tie", createdAt: now.Add(4 * time.Second)},

		{id: "b-c0", parentID: "b", title: "B child 0", createdAt: now.Add(10 * time.Second)},
		{id: "b-c1", parentID: "b", title: "B child 1", createdAt: now.Add(11 * time.Second)},

		{id: "c-c0", parentID: "c", title: "C child 0", createdAt: now.Add(20 * time.Second)},
		{id: "c-c1", parentID: "c", title: "C child 1", createdAt: now.Add(21 * time.Second)},
		{id: "c-c2", parentID: "c", title: "C child 2", createdAt: now.Add(22 * time.Second)},
		{id: "c-c3", parentID: "c", title: "C child 3", createdAt: now.Add(23 * time.Second)},
		{id: "c-c4", parentID: "c", title: "C child 4", createdAt: now.Add(24 * time.Second)},
		{id: "c-c5", parentID: "c", title: "C child 5", createdAt: now.Add(25 * time.Second)},

		{id: "c-c5-g", parentID: "c-c5", title: "C5 grandchild", createdAt: now.Add(30 * time.Second)},
	}

	for _, s := range specs {
		item, err := domain.NewActionItem(domain.ActionItemInput{
			ID:        s.id,
			ProjectID: project.ID,
			ParentID:  s.parentID,
			Kind:      domain.KindPlan,
			ColumnID:  column.ID,
			Title:     s.title,
		}, s.createdAt)
		if err != nil {
			t.Fatalf("NewActionItem(%q) error = %v", s.id, err)
		}
		// NewActionItem stamps CreatedAt from `now`; the `now` argument
		// already supplies the per-spec value, so item.CreatedAt is correct.
		if err := repo.CreateActionItem(ctx, item); err != nil {
			t.Fatalf("CreateActionItem(%q) error = %v", s.id, err)
		}
	}

	return repo, project.ID
}

// TestResolveDottedAddress_Success exercises the happy paths: single-level,
// two-level, three-level, slug-prefix valid, leading-zero positions, and the
// same-CreatedAt UUID tie-breaker.
func TestResolveDottedAddress_Success(t *testing.T) {
	repo, projectID := dottedFixture(t)
	ctx := context.Background()

	// project.Slug for "Tillsyn Demo" → normalizeSlug → "tillsyn-demo".
	slug := "tillsyn-demo"

	cases := []struct {
		name   string
		dotted string
		wantID string
	}{
		{name: "single level zero", dotted: "0", wantID: "a"},
		{name: "single level two", dotted: "2", wantID: "c"},
		{name: "two level", dotted: "1.0", wantID: "b-c0"},
		{name: "two level second", dotted: "1.1", wantID: "b-c1"},
		{name: "two level large", dotted: "2.5", wantID: "c-c5"},
		{name: "three level", dotted: "2.5.0", wantID: "c-c5-g"},
		{name: "slug-prefixed valid single", dotted: slug + ":0", wantID: "a"},
		{name: "slug-prefixed valid three-level", dotted: slug + ":2.5.0", wantID: "c-c5-g"},
		// Tie-break: d-tie-aaa and e-tie-zzz share CreatedAt; id ASC selects
		// d-tie-aaa at position 3 and e-tie-zzz at position 4.
		{name: "tie-break first", dotted: "3", wantID: "d-tie-aaa"},
		{name: "tie-break second", dotted: "4", wantID: "e-tie-zzz"},
		// Leading-zero accepted per strconv.Atoi: "007" → 7. The fixture has
		// no L1[7]; this case is covered separately under not-found tests.
		// Below: leading-zero parses as a valid index when in range.
		{name: "leading-zero zero", dotted: "00", wantID: "a"},
		{name: "leading-zero two", dotted: "02", wantID: "c"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveDottedAddress(ctx, repo, projectID, tc.dotted)
			if err != nil {
				t.Fatalf("ResolveDottedAddress(%q) error = %v, want nil", tc.dotted, err)
			}
			if got != tc.wantID {
				t.Fatalf("ResolveDottedAddress(%q) = %q, want %q", tc.dotted, got, tc.wantID)
			}
		})
	}
}

// TestResolveDottedAddress_NotFound exercises out-of-range indices at every
// level (level-1, level-2 under an existing parent, level-3 under a leaf).
func TestResolveDottedAddress_NotFound(t *testing.T) {
	repo, projectID := dottedFixture(t)
	ctx := context.Background()

	cases := []struct {
		name   string
		dotted string
	}{
		{name: "level-1 out of range", dotted: "99"},
		{name: "level-1 leading-zero out of range", dotted: "007"},
		{name: "level-2 out of range under b (only 2 children)", dotted: "1.5"},
		{name: "level-2 out of range under c (6 children)", dotted: "2.6"},
		{name: "level-3 out of range under c-c5-g (leaf)", dotted: "2.5.0.0"},
		{name: "level-2 under leaf a (no children)", dotted: "0.0"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveDottedAddress(ctx, repo, projectID, tc.dotted)
			if !errors.Is(err, ErrDottedAddressNotFound) {
				t.Fatalf("ResolveDottedAddress(%q) error = %v, want ErrDottedAddressNotFound", tc.dotted, err)
			}
		})
	}
}

// TestResolveDottedAddress_InvalidSyntax exercises shape failures: empty body,
// trailing/leading dots, double dots, non-digit body, malformed slug, slug
// mismatch, and UUID-style input (the resolver expects dotted form — UUID-vs
// detection is a 2.11 caller-side concern).
func TestResolveDottedAddress_InvalidSyntax(t *testing.T) {
	repo, projectID := dottedFixture(t)
	ctx := context.Background()

	cases := []struct {
		name   string
		dotted string
	}{
		{name: "empty", dotted: ""},
		{name: "trailing dot", dotted: "1."},
		{name: "leading dot", dotted: ".1"},
		{name: "double dot", dotted: "1..2"},
		{name: "non-digit", dotted: "abc"},
		{name: "leading-dash", dotted: "-1"},
		{name: "negative segment", dotted: "1.-1.2"},
		{name: "deep nested non-digit", dotted: "1.2.x"},
		{name: "trailing whitespace inside body", dotted: "1. 2"},
		{name: "uuid-style input", dotted: "a5e87c34-3456-4663-9f32-df1b46929e30"},
		{name: "uuid with prefix-looking colon", dotted: "tillsyn-demo:a5e87c34-3456-4663-9f32-df1b46929e30"},
		{name: "slug-prefix empty body", dotted: "tillsyn-demo:"},
		{name: "slug-prefix empty slug", dotted: ":1.2"},
		{name: "slug-prefix bad slug capital", dotted: "Tillsyn:1"},
		{name: "slug-prefix bad slug underscore", dotted: "tillsyn_demo:1"},
		{name: "slug-prefix mismatch", dotted: "wrong-slug:0"},
		{name: "slug-prefix double colon", dotted: "tillsyn-demo:1:2"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveDottedAddress(ctx, repo, projectID, tc.dotted)
			if !errors.Is(err, ErrDottedAddressInvalidSyntax) {
				t.Fatalf("ResolveDottedAddress(%q) error = %v, want ErrDottedAddressInvalidSyntax", tc.dotted, err)
			}
		})
	}
}

// TestResolveDottedAddress_EmptyProjectID rejects callers that fail to supply
// a project — the resolver never parses a project component out of the body.
func TestResolveDottedAddress_EmptyProjectID(t *testing.T) {
	repo, _ := dottedFixture(t)
	ctx := context.Background()

	_, err := ResolveDottedAddress(ctx, repo, "", "0")
	if !errors.Is(err, ErrDottedAddressInvalidSyntax) {
		t.Fatalf("ResolveDottedAddress empty projectID error = %v, want ErrDottedAddressInvalidSyntax", err)
	}
}

// TestIsLikelyDottedAddress covers the shape gate the MCP/CLI dispatch uses to
// pick between UUID lookup and the dotted-resolver path. The gate is intentionally
// permissive about slug shape — slug-prefix matches normalizeSlug's regex; bodies
// match dottedBodyRegex. Anything else (UUIDs, free strings, malformed dots) is
// false so the caller falls through to UUID parsing.
func TestIsLikelyDottedAddress(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "single digit body", in: "0", want: true},
		{name: "multi-segment body", in: "1.5.2", want: true},
		{name: "slug-prefix bare body", in: "tillsyn:1.5.2", want: true},
		{name: "slug-prefix single digit", in: "tillsyn:0", want: true},
		{name: "leading-zero segment", in: "007.0", want: true},
		{name: "empty string", in: "", want: false},
		{name: "whitespace only", in: "   ", want: false},
		{name: "leading dot", in: ".1", want: false},
		{name: "trailing dot", in: "1.", want: false},
		{name: "double dot", in: "1..2", want: false},
		{name: "non-digit body", in: "abc", want: false},
		{name: "uuid", in: "11111111-1111-1111-1111-111111111111", want: false},
		{name: "slug-prefix empty body", in: "tillsyn:", want: false},
		{name: "slug-prefix invalid slug uppercase", in: "Tillsyn:1.5", want: false},
		{name: "slug-prefix invalid slug leading dash", in: "-tillsyn:1.5", want: false},
		{name: "slug-prefix invalid body", in: "tillsyn:1.", want: false},
		{name: "double colon", in: "tillsyn::1.5", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsLikelyDottedAddress(tc.in); got != tc.want {
				t.Fatalf("IsLikelyDottedAddress(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestSplitDottedSlugPrefix covers slug-prefix extraction. The helper is
// intentionally lenient — it returns whatever precedes the first colon as-is,
// leaving deeper validation to the resolver itself.
func TestSplitDottedSlugPrefix(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "no colon", in: "1.5.2", want: ""},
		{name: "uuid passes through", in: "11111111-1111-1111-1111-111111111111", want: ""},
		{name: "slug-prefix", in: "tillsyn:1.5.2", want: "tillsyn"},
		{name: "empty slug-prefix is empty", in: ":1.5.2", want: ""},
		{name: "trimmed leading whitespace", in: "  tillsyn:1.5", want: "tillsyn"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SplitDottedSlugPrefix(tc.in); got != tc.want {
				t.Fatalf("SplitDottedSlugPrefix(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestValidateActionItemIDForMutation enforces mutations-require-UUID for the
// MCP and CLI mutation gates. UUID input passes; dotted (with or without slug)
// returns ErrMutationsRequireUUID; empty input returns ErrDottedAddressInvalidSyntax
// so callers can distinguish missing input from wrong-shape input.
func TestValidateActionItemIDForMutation(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr error
	}{
		{name: "valid UUID accepted", in: "11111111-1111-1111-1111-111111111111", wantErr: nil},
		{name: "dotted body rejected", in: "1.5.2", wantErr: ErrMutationsRequireUUID},
		{name: "slug-prefix dotted rejected", in: "tillsyn:1.5.2", wantErr: ErrMutationsRequireUUID},
		{name: "free-form string rejected", in: "abc", wantErr: ErrMutationsRequireUUID},
		{name: "empty input invalid-syntax", in: "", wantErr: ErrDottedAddressInvalidSyntax},
		{name: "whitespace-only invalid-syntax", in: "   ", wantErr: ErrDottedAddressInvalidSyntax},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateActionItemIDForMutation(tc.in)
			switch {
			case tc.wantErr == nil && err != nil:
				t.Fatalf("ValidateActionItemIDForMutation(%q) error = %v, want nil", tc.in, err)
			case tc.wantErr != nil && !errors.Is(err, tc.wantErr):
				t.Fatalf("ValidateActionItemIDForMutation(%q) error = %v, want %v", tc.in, err, tc.wantErr)
			}
		})
	}
}

// TestResolveActionItemID_UUIDPassesThrough verifies UUID input bypasses the
// dotted resolver entirely — repo is never queried because no dotted walk
// happens.
func TestResolveActionItemID_UUIDPassesThrough(t *testing.T) {
	repo, projectID := dottedFixture(t)
	ctx := context.Background()
	uuid := "11111111-1111-1111-1111-111111111111"

	got, err := ResolveActionItemID(ctx, repo, projectID, uuid)
	if err != nil {
		t.Fatalf("ResolveActionItemID(uuid) error = %v, want nil", err)
	}
	if got != uuid {
		t.Fatalf("ResolveActionItemID(uuid) = %q, want %q", got, uuid)
	}
}

// TestResolveActionItemID_DottedDelegates verifies dotted input flows through
// to ResolveDottedAddress and returns the same UUID a direct ResolveDottedAddress
// call would have returned.
func TestResolveActionItemID_DottedDelegates(t *testing.T) {
	repo, projectID := dottedFixture(t)
	ctx := context.Background()

	want, err := ResolveDottedAddress(ctx, repo, projectID, "0")
	if err != nil {
		t.Fatalf("ResolveDottedAddress baseline error = %v", err)
	}
	got, err := ResolveActionItemID(ctx, repo, projectID, "0")
	if err != nil {
		t.Fatalf("ResolveActionItemID(dotted) error = %v", err)
	}
	if got != want {
		t.Fatalf("ResolveActionItemID(dotted) = %q, want %q", got, want)
	}
}

// TestResolveActionItemID_EmptyInputRejected covers caller-side empty input —
// distinct error class from "shape mismatch" so callers can route the two
// failure modes differently.
func TestResolveActionItemID_EmptyInputRejected(t *testing.T) {
	repo, projectID := dottedFixture(t)
	ctx := context.Background()

	_, err := ResolveActionItemID(ctx, repo, projectID, "   ")
	if !errors.Is(err, ErrDottedAddressInvalidSyntax) {
		t.Fatalf("ResolveActionItemID empty error = %v, want ErrDottedAddressInvalidSyntax", err)
	}
}
