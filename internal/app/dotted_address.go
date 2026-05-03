package app

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

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
