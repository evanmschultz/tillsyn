// useActionItems is the data-layer Solid hook that fetches the action items
// belonging to the currently-selected project via the Wails IPC bridge
// (window.go.main.App.ListActionItems). UI consumers wire this into the
// kanban / list views in a follow-up droplet (Drop FE 2.6 D4); this droplet
// ships pure data plumbing with no rendered surface.
//
// Reactive shape:
//   1. Subscribes to the cross-island `selectedProjectId` Nano Store via
//      @nanostores/solid's `useStore`. The atom is `string | undefined` —
//      `undefined` means "no project selected" (single source of truth, see
//      stores/selection.ts).
//   2. Source signal returns `undefined` whenever (a) Solid is running on the
//      server (`isServer === true`, matches ProjectList's SSR guard) OR (b)
//      no project is selected. createResource treats a falsy/undefined source
//      as "stay pending; do not fire fetcher" — so the hook avoids any IPC
//      call entirely until both client-side AND a project ID exist.
//   3. Fetcher invokes `window.go.main.App.ListActionItems(projectID)`. The
//      Go side (ui/main.go:84-108) trims whitespace and returns `[]` for empty
//      input — but we already guard at the source signal, so the fetcher only
//      ever sees a non-empty trimmed-by-caller projectID.
//
// SSR safety:
//   The source signal explicitly returns `undefined` when `isServer` is true.
//   Without this guard, Astro's SSR pass would attempt to evaluate the
//   fetcher server-side where `window.go` does not exist, throwing at module
//   resolve time. The same pattern keeps SSR + client-initial hydration on
//   the same "pending" state to avoid Solid hydration-mismatch errors (see
//   the long explanation in components/ProjectList.tsx for the deeper
//   rationale around `state === "ready" || "errored"` gating in consumers).
//
// Drop FE 2.6 D3.
import { createResource, type Resource } from "solid-js";
import { isServer } from "solid-js/web";
import { useStore } from "@nanostores/solid";
import { selectedProjectId } from "../stores/selection";

/**
 * ActionItem mirrors the Go-side ActionItemDTO (ui/types.go:33-43) field-for-
 * field. Every field is `string` because the underlying domain enum types
 * (Kind, Role, StructuralType, LifecycleState, Priority) all carry an
 * underlying `string` representation and round-trip lossless as their raw
 * enum-token value (e.g. "build", "builder", "droplet", "in_progress",
 * "high"). When a future drop projects additional ActionItemDTO columns into
 * the wire format, extend both ui/types.go and this interface in lockstep.
 */
export interface ActionItem {
  ID: string;
  ProjectID: string;
  ParentID: string;
  Title: string;
  Kind: string;
  Role: string;
  StructuralType: string;
  LifecycleState: string;
  Priority: string;
}

/**
 * useActionItems returns a Solid `Resource<ActionItem[]>` that tracks the
 * currently-selected project (from the `selectedProjectId` Nano Store) and
 * fetches the action items for that project via Wails IPC.
 *
 * Behavior matrix:
 *   - `isServer === true` → source signal is `undefined`, resource stays
 *     pending, no fetcher invocation.
 *   - `selectedProjectId === undefined` (no project picked yet on the client)
 *     → source signal is `undefined`, resource stays pending, no IPC call.
 *   - `selectedProjectId === "<uuid>"` → fetcher fires with the project ID.
 *     Resource transitions through pending → ready (or errored). When the
 *     store changes again, Solid re-evaluates the source and refires the
 *     fetcher automatically.
 *
 * Consumers wire UI via the standard Solid pattern:
 *   const items = useActionItems();
 *   <Show when={items.state === "ready" || items.state === "errored"}>...
 *
 * See components/ProjectList.tsx for the hydration-safe consumer pattern.
 */
export function useActionItems(): Resource<ActionItem[]> {
  const projectId = useStore(selectedProjectId);
  const [items] = createResource<ActionItem[], string>(
    () => (isServer ? undefined : projectId()),
    async (id) => window.go.main.App.ListActionItems(id),
  );
  return items;
}
