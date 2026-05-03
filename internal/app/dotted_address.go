package app

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/google/uuid"
)

// ErrMutationsRequireUUID is returned when a mutation receives a dotted address
// where the action_item_id argument was supplied. Mutations require the stable
// UUID — dotted addresses are positional and shift under sibling reordering, so
// allowing them on mutations would let a caller silently mutate the wrong item.
var ErrMutationsRequireUUID = errors.New("mutations require UUID action_item_id")

// ErrDottedAddressNotFound is returned when any level of a dotted address
// names an index outside the range of the parent's children at that level.
// ErrDottedAddressInvalidSyntax is returned when the dotted body or its
// optional slug-prefix fails the documented shape check.
var (
	ErrDottedAddressNotFound      = errors.New("dotted address not found")
	ErrDottedAddressInvalidSyntax = errors.New("dotted address invalid syntax")
)

// dottedBodyRegex matches the project-LESS dotted body. Form is one or more
// dot-separated decimal positions: `0`, `2.5`, `1.5.2`. Empty bodies, leading
// dot, trailing dot, and double dots all fail this regex.
var dottedBodyRegex = regexp.MustCompile(`^\d+(\.\d+)*$`)

// dottedSlugRegex matches the slug component of the optional `<slug>:<body>`
// CLI shorthand. Slugs are produced by domain.normalizeSlug — lowercase a-z,
// 0-9, and `-` only, with no leading or trailing dashes.
var dottedSlugRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// IsLikelyDottedAddress reports whether s looks like a dotted address — either
// a bare body matching dottedBodyRegex (e.g. `1.5.2`) or a slug-prefixed form
// `<slug>:<body>` (e.g. `tillsyn:1.5.2`). It performs shape detection only and
// does NOT verify that the slug names a real project or that the body resolves
// against any tree. Callers use this to dispatch between UUID lookup and the
// full ResolveDottedAddress path. Empty input returns false.
func IsLikelyDottedAddress(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	body := s
	if i := strings.Index(s, ":"); i >= 0 {
		slug := s[:i]
		body = s[i+1:]
		if slug == "" || body == "" {
			return false
		}
		if !dottedSlugRegex.MatchString(slug) {
			return false
		}
	}
	return dottedBodyRegex.MatchString(body)
}

// SplitDottedSlugPrefix returns the slug component of a `<slug>:<body>` dotted
// address, or empty string if dotted is bare (no slug prefix). Callers use this
// to map a slug back to a projectID before calling ResolveDottedAddress. The
// returned slug is NOT validated against dottedSlugRegex — callers either pass
// the result to a slug→project lookup that fails on unknown slugs, or rely on
// ResolveDottedAddress to re-validate the slug shape during resolution.
func SplitDottedSlugPrefix(dotted string) string {
	dotted = strings.TrimSpace(dotted)
	if i := strings.Index(dotted, ":"); i >= 0 {
		return dotted[:i]
	}
	return ""
}

// ResolveDottedAddress walks the action-item tree under projectID by the
// 0-indexed positional segments of dotted and returns the matching action
// item's UUID. The resolver requires projectID from the caller — the dotted
// body never carries a project component. The optional `<slug>:<body>` CLI
// shorthand is accepted: the slug is parsed, verified against projectID
// (the project's domain.Project.Slug must equal the supplied slug), and the
// body is then resolved.
//
// Ordering at every level is repo.ListActionItemsByParent's contract:
// created_at ASC, id ASC — UUID id is a globally unique tie-breaker, so the
// ordering is total and ambiguity is unreachable. Position N in that ordering
// IS the dotted index. Empty parentID at the entry level resolves the
// project's level-1 children.
//
// Returns ErrDottedAddressInvalidSyntax for shape failures (empty body,
// leading/trailing/double dots, non-digit body, malformed slug, slug
// mismatch with projectID's slug). Returns ErrDottedAddressNotFound when
// any level's index falls outside the listing at that level. Returns the
// matched action item's UUID on success.
func ResolveDottedAddress(ctx context.Context, repo Repository, projectID, dotted string) (string, error) {
	projectID = strings.TrimSpace(projectID)
	dotted = strings.TrimSpace(dotted)

	if projectID == "" {
		return "", fmt.Errorf("%w: project id required", ErrDottedAddressInvalidSyntax)
	}
	if dotted == "" {
		return "", fmt.Errorf("%w: empty dotted address", ErrDottedAddressInvalidSyntax)
	}

	body := dotted
	if strings.Contains(dotted, ":") {
		parts := strings.SplitN(dotted, ":", 2)
		slug := parts[0]
		body = parts[1]
		if slug == "" || body == "" {
			return "", fmt.Errorf("%w: malformed slug-prefix %q", ErrDottedAddressInvalidSyntax, dotted)
		}
		if !dottedSlugRegex.MatchString(slug) {
			return "", fmt.Errorf("%w: invalid slug %q", ErrDottedAddressInvalidSyntax, slug)
		}
		project, err := repo.GetProject(ctx, projectID)
		if err != nil {
			return "", fmt.Errorf("look up project %q: %w", projectID, err)
		}
		if project.Slug != slug {
			return "", fmt.Errorf("%w: slug %q does not match project %q (slug %q)", ErrDottedAddressInvalidSyntax, slug, projectID, project.Slug)
		}
	}

	if !dottedBodyRegex.MatchString(body) {
		return "", fmt.Errorf("%w: invalid body %q", ErrDottedAddressInvalidSyntax, body)
	}

	segments := strings.Split(body, ".")
	indices := make([]int, 0, len(segments))
	for i, seg := range segments {
		n, err := strconv.Atoi(seg)
		if err != nil || n < 0 {
			return "", fmt.Errorf("%w: segment %d (%q) is not a non-negative decimal", ErrDottedAddressInvalidSyntax, i, seg)
		}
		indices = append(indices, n)
	}

	parentID := ""
	var item domain.ActionItem
	for level, idx := range indices {
		children, err := repo.ListActionItemsByParent(ctx, projectID, parentID)
		if err != nil {
			return "", fmt.Errorf("list action items at level %d (parent %q): %w", level, parentID, err)
		}
		if idx >= len(children) {
			return "", fmt.Errorf("%w: level %d index %d out of range (have %d children of parent %q)", ErrDottedAddressNotFound, level, idx, len(children), parentID)
		}
		item = children[idx]
		parentID = item.ID
	}

	return item.ID, nil
}

// ResolveActionItemID accepts an action-item identifier in either UUID or
// dotted form and returns the canonical UUID. UUID-shaped input is returned
// unchanged after validation; dotted input is resolved via ResolveDottedAddress
// using the supplied projectID. Empty input returns ErrDottedAddressInvalidSyntax
// to surface caller misuse cleanly. Slug-prefix detection is the caller's
// responsibility — the caller resolves slug→projectID and passes the projectID
// into this function (the dotted body, with or without slug-prefix, is then
// handled by ResolveDottedAddress).
func ResolveActionItemID(ctx context.Context, repo Repository, projectID, idOrDotted string) (string, error) {
	idOrDotted = strings.TrimSpace(idOrDotted)
	if idOrDotted == "" {
		return "", fmt.Errorf("%w: empty action_item_id", ErrDottedAddressInvalidSyntax)
	}
	if _, err := uuid.Parse(idOrDotted); err == nil {
		return idOrDotted, nil
	}
	return ResolveDottedAddress(ctx, repo, projectID, idOrDotted)
}

// ValidateActionItemIDForMutation enforces the mutations-require-UUID rule.
// The supplied id must parse as a UUID — a dotted address (with or without a
// slug prefix) returns ErrMutationsRequireUUID. Mutations cannot accept dotted
// form because positional addresses shift under sibling reordering, which would
// let a caller silently mutate the wrong item. Empty input returns
// ErrDottedAddressInvalidSyntax to keep "missing argument" distinct from
// "wrong argument shape" at the boundary.
func ValidateActionItemIDForMutation(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: empty action_item_id", ErrDottedAddressInvalidSyntax)
	}
	if _, err := uuid.Parse(id); err == nil {
		return nil
	}
	return fmt.Errorf("%w: action_item_id %q must be a UUID, not a dotted address", ErrMutationsRequireUUID, id)
}

// GetProjectBySlug returns the project whose slug equals the supplied value.
// It is a thin wrapper over the repository so adapters and CLI code can resolve
// the slug-prefix shorthand (e.g. `tillsyn:1.5.2`) to a projectID before
// calling ResolveActionItemID. Returns ErrNotFound if no project matches.
func (s *Service) GetProjectBySlug(ctx context.Context, slug string) (domain.Project, error) {
	if s == nil {
		return domain.Project{}, errors.New("service is nil")
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return domain.Project{}, fmt.Errorf("%w: empty project slug", ErrDottedAddressInvalidSyntax)
	}
	project, err := s.repo.GetProjectBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return domain.Project{}, fmt.Errorf("%w: project slug %q not found", ErrNotFound, slug)
		}
		return domain.Project{}, fmt.Errorf("get project by slug %q: %w", slug, err)
	}
	return project, nil
}

// ResolveActionItemID delegates to the package-level resolver, supplying the
// service's repository. See ResolveActionItemID (package-level) for full
// semantics.
func (s *Service) ResolveActionItemID(ctx context.Context, projectID, idOrDotted string) (string, error) {
	if s == nil {
		return "", errors.New("service is nil")
	}
	return ResolveActionItemID(ctx, s.repo, projectID, idOrDotted)
}
