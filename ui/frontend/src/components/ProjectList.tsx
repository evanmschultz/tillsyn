// MIGRATION TARGET: @hylla/stil-solid
//
// ProjectList renders the non-archived projects returned by the Go service
// via the Wails in-process bridge (window.go.main.App.ListProjects). Plain
// <ul><li> markup for D1.5; visual polish moves to the @hylla/stil-solid
// component library in a later drop (see REVISION_BRIEF.md §3 Migration
// Targets).
import { createResource, For, Show } from 'solid-js';

type Project = { ID: string; Name: string };

async function fetchProjects(): Promise<Project[]> {
  // SSR guard: Astro static-builds this island server-side to produce initial
  // markup, but the Wails-injected window.go bridge only exists in the browser.
  // Return an empty array during SSR; the resource refires on client hydration
  // where window.go is defined. Without this guard, build-time SSR throws
  // "window is not defined" and Astro serializes the error into the
  // resumability stream — non-fatal but noisy.
  if (typeof window === 'undefined') {
    return [];
  }
  return window.go.main.App.ListProjects();
}

export default function ProjectList() {
  const [projects] = createResource<Project[]>(fetchProjects);

  return (
    <section>
      <h2>Projects</h2>
      <Show
        when={!projects.loading}
        fallback={<p>Loading…</p>}
      >
        <Show
          when={!projects.error}
          fallback={<p role="alert">Error: {String(projects.error)}</p>}
        >
          <Show
            when={(projects() ?? []).length > 0}
            fallback={<p>No projects yet</p>}
          >
            <ul>
              <For each={projects()}>
                {(project) => (
                  <li>
                    {project.ID} — {project.Name}
                  </li>
                )}
              </For>
            </ul>
          </Show>
        </Show>
      </Show>
    </section>
  );
}
