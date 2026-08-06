import {createApp} from 'vue';
import type {Component} from 'vue';

function parseJSON<T>(el: Element, attribute: string, fallback: T): T {
  const value = el.getAttribute(attribute);
  if (!value) return fallback;
  try {
    return JSON.parse(value) as T;
  } catch (err) {
    console.error(`Invalid organization project data in ${attribute}`, err);
    return fallback;
  }
}

async function mount(selector: string, loader: () => Promise<{default: Component}>, props: (el: Element) => Record<string, unknown>) {
  const elements = document.querySelectorAll(selector);
  if (!elements.length) return;
  const {default: component} = await loader();
  for (const el of elements) createApp(component, props(el)).mount(el);
}

export async function initOrgProject() {
  await Promise.all([
    mount('[data-org-project-form]', () => import('../components/orgproject/ProjectForm.vue'), (el) => ({
      schema: parseJSON(el, 'data-schema', {schema_version: 1, fields: [], list_view: {columns: []}, filters: [], metrics: []}),
      initialValues: parseJSON(el, 'data-values', {}),
      members: parseJSON(el, 'data-members', []),
    })),
    mount('[data-org-project-list]', () => import('../components/orgproject/ProjectList.vue'), (el) => ({
      rows: parseJSON(el, 'data-rows', []),
      baseLink: el.getAttribute('data-base-link') || '',
      archivedLabel: el.getAttribute('data-archived-label') || 'Archived',
      noDescription: el.getAttribute('data-no-description') || '',
    })),
    mount('[data-org-project-dashboard]', () => import('../components/orgproject/ProjectDashboard.vue'), (el) => ({
      metrics: parseJSON(el, 'data-metrics', []),
    })),
    mount('[data-org-project-history]', () => import('../components/orgproject/ProjectHistory.vue'), (el) => ({
      changes: parseJSON(el, 'data-changes', []),
      emptyText: el.getAttribute('data-empty-text') || '',
    })),
    mount('[data-org-project-config-editor]', () => import('../components/orgproject/ConfigEditor.vue'), (el) => ({
      schema: parseJSON(el, 'data-schema', {schema_version: 1, fields: [], list_view: {columns: []}, filters: [], metrics: []}),
    })),
    mount('[data-org-project-editor-teams]', () => import('../components/orgproject/EditorTeamsEditor.vue'), (el) => ({
      teams: parseJSON(el, 'data-teams', []),
      emptyText: el.getAttribute('data-empty-text') || '',
    })),
    mount('[data-org-project-activity]', () => import('../components/orgproject/ProjectActivity.vue'), (el) => ({
      summary: parseJSON(el, 'data-summary', {since: '', repositories: [], commits: [], open_pulls: 0, merged_pulls: 0, release_count: 0}),
      locale: {
        openPulls: el.getAttribute('data-locale-open-pulls') || '',
        mergedPulls: el.getAttribute('data-locale-merged-pulls') || '',
        releases: el.getAttribute('data-locale-releases') || '',
        repositories: el.getAttribute('data-locale-repositories') || '',
        repository: el.getAttribute('data-locale-repository') || '',
        recentCommits: el.getAttribute('data-locale-recent-commits') || '',
        latestRelease: el.getAttribute('data-locale-latest-release') || '',
        empty: el.getAttribute('data-locale-empty') || '',
        since: el.getAttribute('data-locale-since') || '',
      },
    })),
  ]);
}
