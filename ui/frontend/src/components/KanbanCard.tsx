// MIGRATION TARGET: @hylla/stil-solid
//
// KanbanCard — SolidJS island leaf consumed by KanbanBoard.tsx's reactive `<For>`.
// Renders one ActionItem; no IPC, no resource, no client state. Not authored as
// `.astro` because Astro components don't render inside hydrated framework islands
// at runtime — see Phase 2A plan zero-JS justification.

import { Show } from "solid-js";
import type { ActionItem } from "../hooks/useActionItems";

/**
 * structuralGlyph maps cascade structural_type to a visual marker.
 * Kept inline (not exported) to stay within 1-block atomicity.
 */
function structuralGlyph(type: string): string {
  switch (type) {
    case "drop":
      return "↓";
    case "segment":
      return "▱";
    case "confluence":
      return "▼";
    case "droplet":
      return "○";
    default:
      return "";
  }
}

interface KanbanCardProps {
  item: ActionItem;
}

/**
 * KanbanCard renders a single action item as a card within the kanban board.
 * Displays title, structural-type glyph, kind badge, optional role badge, and
 * priority indicator. Pure render component — no internal state, no event handlers.
 */
export default function KanbanCard(props: KanbanCardProps) {
  return (
    <li
      class="kanban-card"
      tabindex={0}
      aria-label={`${props.item.Title} (${props.item.Kind})`}
    >
      <h3 class="kanban-card-title">{props.item.Title}</h3>
      <div class="kanban-card-meta">
        <span class="kanban-card-glyph" aria-hidden="true">
          {structuralGlyph(props.item.StructuralType)}
        </span>
        <span class="kanban-card-badge">{props.item.Kind}</span>
        <Show when={props.item.Role}>
          <span class="kanban-card-badge">{props.item.Role}</span>
        </Show>
        <span
          class="kanban-card-priority"
          role="img"
          data-priority={props.item.Priority}
          aria-label={`Priority: ${props.item.Priority}`}
        />
      </div>
    </li>
  );
}
