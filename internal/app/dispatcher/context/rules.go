package context

import (
	stdcontext "context"
	"fmt"
	"sort"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// resolveParent renders the parent action item's identity + key fields into
// a markdown block. Returns empty content (not an error) when the supplied
// item has no parent — the rule is "render the parent IF one exists."
func resolveParent(
	ctx stdcontext.Context,
	item domain.ActionItem,
	reader ActionItemReader,
) ([]byte, error) {
	if strings.TrimSpace(item.ParentID) == "" {
		return nil, nil
	}
	parent, err := reader.GetActionItem(ctx, item.ParentID)
	if err != nil {
		return nil, fmt.Errorf("get parent %s: %w", item.ParentID, err)
	}
	return renderActionItemBlock(parent), nil
}

// resolveParentGitDiff captures `git diff <parent.start_commit>..<parent.end_commit>`
// when the parent has both commit anchors populated. When either anchor is
// empty (or the parent itself is missing), returns empty content with no
// error — the rule is "if observable, give it to me."
func resolveParentGitDiff(
	ctx stdcontext.Context,
	item domain.ActionItem,
	reader ActionItemReader,
	diff GitDiffReader,
) ([]byte, error) {
	if strings.TrimSpace(item.ParentID) == "" {
		return nil, nil
	}
	parent, err := reader.GetActionItem(ctx, item.ParentID)
	if err != nil {
		return nil, fmt.Errorf("get parent %s: %w", item.ParentID, err)
	}
	start := strings.TrimSpace(parent.StartCommit)
	end := strings.TrimSpace(parent.EndCommit)
	if start == "" || end == "" {
		return nil, nil
	}
	out, err := diff.Diff(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("diff %s..%s: %w", start, end, err)
	}
	return out, nil
}

// resolveSiblingsByKind lists same-parent action items, filters by kind,
// emits the LATEST round only (per master PLAN.md F.7.18 spec) when multiple
// siblings share a kind. Renders each surviving sibling as a markdown block.
//
// "Latest round" is determined by CreatedAt (most recent wins). Ties on
// CreatedAt fall through to ID (lexicographic) to keep ordering deterministic
// across runs — important for the eventual `parent_git_diff` correlation
// that compares spawn-bundle hashes across cascade rounds.
func resolveSiblingsByKind(
	ctx stdcontext.Context,
	item domain.ActionItem,
	acceptedKinds []domain.Kind,
	reader ActionItemReader,
) ([]byte, error) {
	if strings.TrimSpace(item.ParentID) == "" {
		return nil, nil
	}
	siblings, err := reader.ListSiblings(ctx, item.ParentID)
	if err != nil {
		return nil, fmt.Errorf("list siblings of %s: %w", item.ParentID, err)
	}

	// Group filtered siblings by kind, picking the latest round per kind.
	latestByKind := map[domain.Kind]domain.ActionItem{}
	for _, sib := range siblings {
		// Don't render the spawning item itself among its siblings.
		if sib.ID == item.ID {
			continue
		}
		if !kindMatches(sib.Kind, acceptedKinds) {
			continue
		}
		existing, present := latestByKind[sib.Kind]
		if !present {
			latestByKind[sib.Kind] = sib
			continue
		}
		// Latest = most recent CreatedAt; tie-break on ID lexicographically.
		if sib.CreatedAt.After(existing.CreatedAt) {
			latestByKind[sib.Kind] = sib
		} else if sib.CreatedAt.Equal(existing.CreatedAt) && sib.ID > existing.ID {
			latestByKind[sib.Kind] = sib
		}
	}

	if len(latestByKind) == 0 {
		return nil, nil
	}

	// Render in the declaration order of acceptedKinds so the bundle ordering
	// matches the template's stated preference.
	var buf strings.Builder
	first := true
	for _, kind := range acceptedKinds {
		sib, ok := latestByKind[kind]
		if !ok {
			continue
		}
		if !first {
			buf.WriteString("\n\n")
		}
		first = false
		buf.Write(renderActionItemBlock(sib))
	}
	return []byte(buf.String()), nil
}

// resolveAncestorsByKind walks up the parent chain and renders the FIRST
// ancestor whose Kind matches an entry in acceptedKinds. The walk halts on
// the first match per the F.7.18 spec — adopters who want every ancestor
// declare descendants_by_kind on the parent's binding instead.
//
// Walk semantics: starts from item.ParentID and follows ParentID upward. A
// loop guard caps the walk at 256 hops as a defense-in-depth backstop
// against cyclic data; the domain layer prevents cycles at create time, so
// hitting the cap means corrupted SQLite data — surfaces as a wrapped error.
func resolveAncestorsByKind(
	ctx stdcontext.Context,
	item domain.ActionItem,
	acceptedKinds []domain.Kind,
	reader ActionItemReader,
) ([]byte, error) {
	const maxHops = 256
	currentID := strings.TrimSpace(item.ParentID)
	visited := map[string]struct{}{}
	for hop := 0; hop < maxHops; hop++ {
		if currentID == "" {
			return nil, nil
		}
		if _, seen := visited[currentID]; seen {
			return nil, fmt.Errorf("ancestors_by_kind: cycle detected at %s", currentID)
		}
		visited[currentID] = struct{}{}

		ancestor, err := reader.GetActionItem(ctx, currentID)
		if err != nil {
			return nil, fmt.Errorf("get ancestor %s: %w", currentID, err)
		}
		if kindMatches(ancestor.Kind, acceptedKinds) {
			return renderActionItemBlock(ancestor), nil
		}
		currentID = strings.TrimSpace(ancestor.ParentID)
	}
	return nil, fmt.Errorf("ancestors_by_kind: max walk depth %d exceeded", maxHops)
}

// resolveDescendantsByKind walks the cascade subtree depth-first from the
// spawning item and renders every direct + transitive descendant whose Kind
// matches acceptedKinds. Per master PLAN.md F.7.18 there is NO schema rule
// against descendants_by_kind on `kind=plan`; template authors are trusted
// to use it sensibly.
//
// A loop guard caps the walk at 4096 visited nodes — a soft cap that
// protects the engine from runaway data (the domain layer prevents cycles,
// but a deeply-nested adopter tree could still consume the entire wall-
// clock budget without one). Hitting the cap returns a wrapped error so the
// rule fails loudly rather than silently truncating.
func resolveDescendantsByKind(
	ctx stdcontext.Context,
	item domain.ActionItem,
	acceptedKinds []domain.Kind,
	reader ActionItemReader,
) ([]byte, error) {
	const maxNodes = 4096
	type frame struct {
		parentID string
	}
	stack := []frame{{parentID: item.ID}}
	visited := map[string]struct{}{}
	var matches []domain.ActionItem

	for len(stack) > 0 {
		// Outer/inner timeout guard — bail early if the surrounding context
		// is dead so the descendant walk does not race the engine cap.
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if len(visited) >= maxNodes {
			return nil, fmt.Errorf("descendants_by_kind: max walk size %d exceeded", maxNodes)
		}
		if _, seen := visited[f.parentID]; seen {
			continue
		}
		visited[f.parentID] = struct{}{}

		children, err := reader.ListChildren(ctx, f.parentID)
		if err != nil {
			return nil, fmt.Errorf("list children of %s: %w", f.parentID, err)
		}
		for _, child := range children {
			if kindMatches(child.Kind, acceptedKinds) {
				matches = append(matches, child)
			}
			stack = append(stack, frame{parentID: child.ID})
		}
	}

	if len(matches) == 0 {
		return nil, nil
	}

	// Sort matches by (CreatedAt, ID) so the rendering order is deterministic
	// across runs regardless of the adapter's iteration order.
	sort.Slice(matches, func(i, j int) bool {
		if !matches[i].CreatedAt.Equal(matches[j].CreatedAt) {
			return matches[i].CreatedAt.Before(matches[j].CreatedAt)
		}
		return matches[i].ID < matches[j].ID
	})

	var buf strings.Builder
	for i, m := range matches {
		if i > 0 {
			buf.WriteString("\n\n")
		}
		buf.Write(renderActionItemBlock(m))
	}
	return []byte(buf.String()), nil
}

// renderActionItemBlock renders one action item as a compact markdown block
// the spawning agent can read inline. The format is intentionally terse —
// title, kind, paths, packages, completion-notes excerpt — so the per-rule
// truncation cap does not eat the agent's first useful chunk.
//
// Format:
//
//	### <title> (<kind>)
//	id: <id>
//	[paths: <paths>]                — only when non-empty
//	[packages: <packages>]          — only when non-empty
//	[start_commit: <sha>]           — only when non-empty
//	[end_commit: <sha>]             — only when non-empty
//
//	<description>
//
// Description is rendered verbatim (the planner's prose). The dispatcher
// trusts adopters to keep descriptions terse; truncation falls through to
// the per-rule MaxChars cap.
func renderActionItemBlock(item domain.ActionItem) []byte {
	var b strings.Builder
	b.WriteString("### ")
	if item.Title != "" {
		b.WriteString(item.Title)
	} else {
		b.WriteString("(untitled)")
	}
	if item.Kind != "" {
		b.WriteString(" (")
		b.WriteString(string(item.Kind))
		b.WriteString(")")
	}
	b.WriteString("\n")
	b.WriteString("id: ")
	b.WriteString(item.ID)
	b.WriteString("\n")
	if len(item.Paths) > 0 {
		b.WriteString("paths: ")
		b.WriteString(strings.Join(item.Paths, ", "))
		b.WriteString("\n")
	}
	if len(item.Packages) > 0 {
		b.WriteString("packages: ")
		b.WriteString(strings.Join(item.Packages, ", "))
		b.WriteString("\n")
	}
	if item.StartCommit != "" {
		b.WriteString("start_commit: ")
		b.WriteString(item.StartCommit)
		b.WriteString("\n")
	}
	if item.EndCommit != "" {
		b.WriteString("end_commit: ")
		b.WriteString(item.EndCommit)
		b.WriteString("\n")
	}
	if item.Description != "" {
		b.WriteString("\n")
		b.WriteString(item.Description)
		b.WriteString("\n")
	}
	return []byte(b.String())
}
