import {createApp} from 'vue';
import type {Component} from 'vue';
import {attachSearchBox} from '../modules/search.ts';

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

function controlDisplayValue(control: HTMLElement | null): string {
  if (control instanceof HTMLSelectElement) {
    if (!control.value && !control.multiple) return '';
    if (!control.multiple) return Array.from(control.options).find((option) => option.value === control.value)?.textContent?.trim() || '';
    return Array.from(control.selectedOptions, (option) => option.textContent?.trim() || '').filter(Boolean).join(', ');
  }
  if (control instanceof HTMLInputElement) {
    if (control.type === 'checkbox') return control.checked ? control.value : '';
    return control.value.trim();
  }
  if (control instanceof HTMLTextAreaElement) return control.value.trim();
  return '';
}

function refreshOrgProjectCreateSummary(form: HTMLFormElement, summary: HTMLElement) {
  const emptyText = summary.getAttribute('data-empty-text') || '—';
  const rows = summary.querySelectorAll<HTMLElement>('[data-summary-control]');
  let completeCount = 0;
  for (const row of rows) {
    const controlID = row.getAttribute('data-summary-control') || '';
    const candidate = form.querySelector<HTMLElement>(`#${controlID}`);
    const control = candidate && form.contains(candidate) ? candidate : null;
    const value = controlDisplayValue(control);
    row.querySelector<HTMLElement>('[data-summary-value]')!.textContent = value || emptyText;
    row.classList.toggle('is-complete', Boolean(value));
    row.classList.toggle('is-pending', !value);
    if (value) completeCount++;
  }
  summary.querySelector<HTMLElement>('[data-summary-complete-count]')!.textContent = String(completeCount);
  summary.querySelector<HTMLElement>('[data-summary-total-count]')!.textContent = String(rows.length);
}

export function initOrgProjectCreateSummary() {
  for (const form of document.querySelectorAll<HTMLFormElement>('[data-org-project-create-form]')) {
    const summary = form.querySelector<HTMLElement>('[data-org-project-create-summary]');
    if (!summary || form.hasAttribute('data-org-project-create-summary-initialized')) continue;
    form.setAttribute('data-org-project-create-summary-initialized', 'true');
    const refresh = () => refreshOrgProjectCreateSummary(form, summary);
    form.addEventListener('input', refresh);
    form.addEventListener('change', refresh);
    refresh();
  }
}

type RepositorySearchResponse = {
  data: Array<{repository: {id: number, full_name: string}}>,
};

export function initOrgProjectRepositorySearch() {
  for (const container of document.querySelectorAll<HTMLElement>('[data-org-project-repository-search]')) {
    if (container.hasAttribute('data-org-project-repository-search-initialized')) continue;
    const form = container.closest<HTMLFormElement>('form');
    const input = container.querySelector<HTMLInputElement>('input.prompt');
    const repositoryID = form?.querySelector<HTMLInputElement>('input[name="repository_id"]');
    const ownerID = container.getAttribute('data-owner-id');
    if (!form || !input || !repositoryID || !ownerID) continue;

    container.setAttribute('data-org-project-repository-search-initialized', 'true');
    const panel = container.closest<HTMLElement>('.org-project-repository-panel');
    const linkedRepositoryIDs = new Set(Array.from(
      panel?.querySelectorAll<HTMLElement>('[data-org-project-repository-id]') ?? [],
      (item) => item.getAttribute('data-org-project-repository-id'),
    ).filter((id): id is string => Boolean(id)));
    const selectError = container.getAttribute('data-select-error') || 'Please select a repository from the search results.';

    attachSearchBox<RepositorySearchResponse>(
      container,
      `${window.config.appSubUrl}/repo/search?q={query}&uid=${ownerID}`,
      (response) => response.data
        .filter(({repository}) => !linkedRepositoryIDs.has(String(repository.id)))
        .map(({repository}) => ({title: repository.full_name, value: String(repository.id)})),
      {
        onInput: () => {
          repositoryID.value = '';
          input.setCustomValidity('');
        },
        onSelect: (result) => {
          repositoryID.value = result.value || '';
          input.setCustomValidity('');
        },
      },
    );

    form.addEventListener('submit', (event) => {
      if (repositoryID.value) return;
      event.preventDefault();
      input.setCustomValidity(selectError);
      input.reportValidity();
    });
  }
}

export async function initOrgProject() {
  await Promise.all([
    mount('[data-org-project-form]', () => import('../components/orgproject/ProjectForm.vue'), (el) => ({
      schema: parseJSON(el, 'data-schema', {schema_version: 1, fields: [], list_view: {columns: []}, filters: [], metrics: []}),
      initialValues: parseJSON(el, 'data-values', {}),
      members: parseJSON(el, 'data-members', []),
      layout: el.getAttribute('data-layout') || 'plain',
      labels: {
        plan: {title: el.getAttribute('data-plan-title') || '', description: el.getAttribute('data-plan-description') || ''},
        status: {title: el.getAttribute('data-status-title') || '', description: el.getAttribute('data-status-description') || ''},
        other: {title: el.getAttribute('data-other-title') || '', description: el.getAttribute('data-other-description') || ''},
      },
    })),
    mount('[data-org-project-list]', () => import('../components/orgproject/ProjectList.vue'), (el) => ({
      rows: parseJSON(el, 'data-rows', []),
      baseLink: el.getAttribute('data-base-link') || '',
      archivedLabel: el.getAttribute('data-archived-label') || 'Archived',
      noDescription: el.getAttribute('data-no-description') || '',
      projectLabel: el.getAttribute('data-project-label') || 'Project',
      updatedLabel: el.getAttribute('data-updated-label') || 'Updated',
    })),
    mount('[data-org-project-dashboard]', () => import('../components/orgproject/ProjectDashboard.vue'), (el) => ({
      summary: parseJSON(el, 'data-summary', {active: 0, blocked: 0, overdue: 0, due_soon: 0, average_progress: 0}),
      baseLink: el.getAttribute('data-base-link') || '',
      labels: {
        blocked: el.getAttribute('data-label-blocked') || '',
        overdue: el.getAttribute('data-label-overdue') || '',
        dueSoon: el.getAttribute('data-label-due-soon') || '',
        averageProgress: el.getAttribute('data-label-average-progress') || '',
        blockedDescription: el.getAttribute('data-description-blocked') || '',
        overdueDescription: el.getAttribute('data-description-overdue') || '',
        dueSoonDescription: el.getAttribute('data-description-due-soon') || '',
        averageProgressDescription: el.getAttribute('data-description-average-progress') || '',
      },
    })),
    mount('[data-org-project-history]', () => import('../components/orgproject/ProjectHistory.vue'), (el) => ({
      changes: parseJSON(el, 'data-changes', []),
      fieldLabels: parseJSON(el, 'data-field-labels', {}),
      emptyText: el.getAttribute('data-empty-text') || '',
      locale: {
        changed: el.getAttribute('data-locale-changed') || '',
        user: el.getAttribute('data-locale-user') || '',
        source: el.getAttribute('data-locale-source') || '',
        sourceWeb: el.getAttribute('data-locale-source-web') || '',
        sourceApi: el.getAttribute('data-locale-source-api') || '',
        sourceLegacyImport: el.getAttribute('data-locale-source-legacy-import') || '',
        details: el.getAttribute('data-locale-details') || '',
        before: el.getAttribute('data-locale-before') || '',
        after: el.getAttribute('data-locale-after') || '',
        requestID: el.getAttribute('data-locale-request-id') || '',
      },
    })),
    mount('[data-org-project-config-editor]', () => import('../components/orgproject/ConfigEditor.vue'), (el) => ({
      schema: parseJSON(el, 'data-schema', {schema_version: 1, fields: [], list_view: {columns: []}, filters: [], metrics: []}),
      labels: parseJSON(el, 'data-labels', {}),
    })),
    mount('[data-org-project-editor-teams]', () => import('../components/orgproject/EditorTeamsEditor.vue'), (el) => ({
      teams: parseJSON(el, 'data-teams', []),
      emptyText: el.getAttribute('data-empty-text') || '',
    })),
    mount('[data-org-project-activity]', () => import('../components/orgproject/ProjectActivity.vue'), (el) => ({
      summary: parseJSON(el, 'data-summary', {since: '', repositories: [], commits: [], progress: [], open_pulls: 0, merged_pulls: 0, release_count: 0, partial: false}),
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
    mount('[data-org-project-workbench]', () => import('../components/orgproject/ProjectWorkbench.vue'), (el) => ({
      workbench: parseJSON(el, 'data-workbench', {projects: [], people: [], attention: {}, configured_organizations: 0}),
      onlyMine: (el.getAttribute('data-only-mine') || '').trim() === 'true',
      baseLink: (el.getAttribute('data-base-link') || '').trim() || window.location.pathname,
      labels: parseJSON(el, 'data-labels', {}),
    })),
  ]);
  initOrgProjectCreateSummary();
  initOrgProjectRepositorySearch();
}
