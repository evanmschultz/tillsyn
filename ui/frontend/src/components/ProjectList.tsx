// MIGRATION TARGET: @hylla/stil-solid
//
// ProjectList renders the non-archived projects returned by the Go service
// via the Wails in-process bridge (window.go.main.App.ListProjects). Plain
// <ul><li> markup for D1.5; visual polish moves to the @hylla/stil-solid
// component library in a later drop (see REVISION_BRIEF.md §3 Migration
// Targets).
import { createResource, For, Show } from 'solid-js';
import { isServer } from 'solid-js/web';

type Project = { ID: string; Name: string };

async function fetchProjects(): Promise<Project[]> {
  // The Wails-injected window.go bridge only exists in the browser. The
  // createResource source signal below (`() => !isServer`) returns false
  // during Astro SSR, which keeps the resource in its pending state and
  // skips this fetcher entirely server-side. On client hydration the source
  // becomes true and Solid re-evaluates the resource, firing this fetcher
  // for the first time.
  return window.go.main.App.ListProjects();
}

export default function ProjectList() {
  // SSR-aware resource: the source signal is `() => !isServer`. On the
  // server side Solid sees a falsy source and leaves the resource pending
  // (no SSR data serialization, no resolved empty-array baked into the
  // hydration stream). On the client the source becomes truthy and the
  // fetcher fires against the live Wails IPC bridge.
  //
  // Without this guard, a `typeof window === 'undefined'` short-circuit in
  // the fetcher resolves the resource to `[]` server-side. Solid's async-SSR
  // contract then serializes that resolved value into the page; the client
  // reuses the serialized state and never re-fetches, leaving the UI stuck
  // on the empty-state "No projects yet" fallback even when the underlying
  // database has projects. See Solid's solid-ssr docs: "Data is serialized,
  // sent with the page, and reused by the client as needed."
  const [projects] = createResource<Project[], boolean>(
    () => !isServer,
    fetchProjects,
  );

  return (
    <section>
      <h2>Projects</h2>
      {/*
        Outer guard gates on terminal resource states ("ready" + "errored")
        instead of the loading flag. This is load-bearing for SSR-hydration
        match: with the source-signal `() => !isServer`, SSR sees state=
        "unresolved" (loading=false) and the client-initial render sees state=
        "pending" (loading=true). If the outer Show were `when={!projects.
        loading}`, SSR would render the projects branch while the client would
        render the Loading fallback — DOM mismatch → Solid throws Hydration
        Mismatch and the UI stays stuck on whatever SSR painted. Gating on
        terminal states keeps SSR + client-initial both on the "Loading…"
        fallback so hydration's DOM matches.
      */}
      <Show
        when={projects.state === "ready" || projects.state === "errored"}
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
