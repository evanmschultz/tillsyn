// MIGRATION TARGET: @hylla/stil-solid
//
// KanbanBoard — SolidJS island that renders the 3-column kanban layout
// (todo, in_progress, complete, failed) with reactive iteration via <For>.
// Consumes useActionItems() hook for IPC-backed ActionItem data, reads
// selectedProjectId from the Nano Store for reactive project switching.
//
// Mirrors ProjectList.tsx hydration-safe gating pattern:
//   - Source signal from selectedProjectId (via useActionItems hook).
//   - Outer <Show when={state === "ready" || state === "errored"}> gates on
//     terminal resource states to avoid SSR/client hydration mismatch.
//   - Inner error + empty guards with accessible fallbacks.
// See components/ProjectList.tsx for deeper SSR-safety rationale.

import { For, Show } from "solid-js";
import { useActionItems, type ActionItem } from "../hooks/useActionItems";
import KanbanCard from "./KanbanCard";

const LIFECYCLE_STATES = ["todo", "in_progress", "complete", "failed"] as const;

const LIFECYCLE_LABELS: Record<string, string> = {
  todo: "To Do",
  in_progress: "In Progress",
  complete: "Complete",
  failed: "Failed",
};

/**
 * KanbanBoard renders the kanban grid partitioned by lifecycle state.
 * No props — pulls both items and selectedProjectId from hooks + store.
 */
export default function KanbanBoard() {
  const items = useActionItems();

  // Inline partition logic: filter items by lifecycle state.
  const byState = (state: string) => (items() ?? []).filter((i) => i.LifecycleState === state);

  return (
    <section class="kanban-board" aria-label="Action items kanban">
      {/*
        Outer guard gates on terminal states (ready/errored) instead of
        loading flag. This is load-bearing for SSR-hydration match — with
        the source signal `() => isServer ? undefined : projectId()` from
        useActionItems, SSR sees state="unresolved" (loading=false) and
        the client-initial render sees state="pending" (loading=true). If
        the outer Show were `when={!items.loading}`, SSR would render the
        kanban while the client would render the Loading fallback — DOM
        mismatch → Solid throws Hydration Mismatch. Gating on terminal
        states keeps SSR + client-initial both on the "Select a project"
        fallback so hydration's DOM matches.
      */}
      <Show
        when={items.state === "ready" || items.state === "errored"}
        fallback={
          <p class="project-sidebar-status" data-tone="empty">
            Select a project to view its kanban
          </p>
        }
      >
        {/* Error guard with accessible alert semantics. */}
        <Show
          when={!items.error}
          fallback={
            <p role="alert" class="project-sidebar-status" data-tone="error">
              Error loading kanban: {String(items.error)}
            </p>
          }
        >
          {/* Render 4-column kanban grid (responsive via CSS media queries). */}
          <For each={LIFECYCLE_STATES}>
            {(state) => (
              <article class="kanban-column" data-state={state}>
                <header class="kanban-column-header">
                  <span>{LIFECYCLE_LABELS[state]}</span>
                  <span class="kanban-column-count">{byState(state).length}</span>
                </header>

                <ul class="kanban-column-body" role="list">
                  <Show
                    when={byState(state).length > 0}
                    fallback={<li class="kanban-column-empty">No items</li>}
                  >
                    <For each={byState(state)}>{(item) => <KanbanCard item={item} />}</For>
                  </Show>
                </ul>
              </article>
            )}
          </For>
        </Show>
      </Show>
    </section>
  );
}
