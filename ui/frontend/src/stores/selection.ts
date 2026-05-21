/**
 * Cross-island shared selection state.
 *
 * Uses Nano Stores per Astro docs (/recipes/sharing-state-islands):
 * "Traditional UI framework patterns like React context providers do not
 * work when components are partially hydrated within Astro or Markdown.
 * To solve this, Astro recommends using Nano Stores."
 *
 * Module-scope SolidJS createSignal would NOT reliably share state across
 * independently-hydrated client:idle islands (Vite code-splitting may dedupe
 * the module by accident but is not guaranteed). Each island ships as its own
 * client bundle, and a per-island module-scope signal would create N
 * disconnected reactive scopes. Nano Stores is the Astro-blessed cross-island
 * primitive: a single module-scope atom plus framework bindings
 * (@nanostores/solid's useStore) that subscribe per-island to the same store.
 */
import { atom } from "nanostores";

const selectedProjectIdAtom = atom<string | undefined>(undefined);

/**
 * Atom holding the currently-selected project id, or `undefined` when no
 * project is selected. Subscribe via `useStore(selectedProjectId)` from
 * `@nanostores/solid` inside SolidJS components, or via
 * `selectedProjectId.subscribe(...)` for raw subscriptions.
 */
export const selectedProjectId = selectedProjectIdAtom;

/**
 * Sets the selected project. Empty-string input coerces to `undefined` per
 * the single-source-of-truth rule: "no selection" === `undefined`. Callers
 * (e.g. clearing a `<select>` value) can pass `""` without producing a
 * distinct "empty string selected" state separate from "no selection".
 */
export function setSelectedProjectId(value: string | undefined): void {
	if (value === "" || value === undefined) {
		selectedProjectIdAtom.set(undefined);
		return;
	}
	selectedProjectIdAtom.set(value);
}
