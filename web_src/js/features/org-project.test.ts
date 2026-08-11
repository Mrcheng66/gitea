import {initOrgProject, initOrgProjectCreateSummary, initOrgProjectRepositorySearch} from './org-project.ts';
import {GET} from '../modules/fetch.ts';

vi.mock('../modules/fetch.ts', () => ({
  GET: vi.fn(),
}));

describe('organization project workbench', {concurrent: false}, () => {
  test('uses the current path and selects only mine from padded attributes', async () => {
    document.body.innerHTML = `
      <div
        data-org-project-workbench
        data-workbench='{"projects":[],"people":[],"attention":{},"configured_organizations":0}'
        data-only-mine=" true "
        data-base-link=""
        data-labels='{"title":"Workbench","subtitle":"Projects","team":"Team","mine":"Mine","attention":"Attention","empty":"Empty","configure":"Configure"}'
      ></div>
    `;

    await initOrgProject();

    expect(document.querySelector<HTMLAnchorElement>('a[href="/"]')!.classList).not.toContain('active');
    expect(document.querySelector<HTMLAnchorElement>('a[href="/?scope=mine"]')!.classList).toContain('active');
  });

  test('selects the team scope after the mine query is cleared', async () => {
    document.body.innerHTML = `
      <div
        data-org-project-workbench
        data-workbench='{"projects":[],"people":[],"attention":{},"configured_organizations":0}'
        data-only-mine=" false "
        data-base-link=""
        data-labels='{"title":"Workbench","subtitle":"Projects","team":"Team","mine":"Mine","attention":"Attention","empty":"Empty","configure":"Configure"}'
      ></div>
    `;

    await initOrgProject();

    expect(document.querySelector<HTMLAnchorElement>('a[href="/"]')!.classList).toContain('active');
    expect(document.querySelector<HTMLAnchorElement>('a[href="/?scope=mine"]')!.classList).not.toContain('active');
  });
});

describe('organization project create summary', {concurrent: false}, () => {
  test('reads and updates native form controls', () => {
    document.body.innerHTML = `
      <form data-org-project-create-form>
        <input id="slug" value="project-one">
        <select id="org-project-field-stage">
          <option value="planned" selected>Planned</option>
          <option value="development">Development</option>
        </select>
        <select id="org-project-field-followers" multiple>
          <option value="1" selected>Alex</option>
          <option value="2">Sam</option>
        </select>
        <aside data-org-project-create-summary data-empty-text="Not set">
          <span data-summary-complete-count></span>
          <span data-summary-total-count></span>
          <div data-summary-control="slug"><i data-summary-state></i><b data-summary-value></b></div>
          <div data-summary-control="org-project-field-stage"><i data-summary-state></i><b data-summary-value></b></div>
          <div data-summary-control="org-project-field-followers"><i data-summary-state></i><b data-summary-value></b></div>
          <div data-summary-control="missing"><i data-summary-state></i><b data-summary-value></b></div>
        </aside>
      </form>
    `;

    initOrgProjectCreateSummary();

    const summary = document.querySelector<HTMLElement>('[data-org-project-create-summary]')!;
    expect(summary.querySelector('[data-summary-control="slug"] [data-summary-value]')!.textContent).toBe('project-one');
    expect(summary.querySelector('[data-summary-control="org-project-field-stage"] [data-summary-value]')!.textContent).toBe('Planned');
    expect(summary.querySelector('[data-summary-control="org-project-field-followers"] [data-summary-value]')!.textContent).toBe('Alex');
    expect(summary.querySelector('[data-summary-control="missing"] [data-summary-value]')!.textContent).toBe('Not set');
    expect(summary.querySelector('[data-summary-complete-count]')!.textContent).toBe('3');
    expect(summary.querySelector('[data-summary-total-count]')!.textContent).toBe('4');

    const slug = document.querySelector<HTMLInputElement>('#slug')!;
    slug.value = '';
    slug.dispatchEvent(new Event('input', {bubbles: true}));
    expect(summary.querySelector('[data-summary-control="slug"] [data-summary-value]')!.textContent).toBe('Not set');
    expect(summary.querySelector('[data-summary-control="slug"]')!.classList).toContain('is-pending');
    expect(summary.querySelector('[data-summary-complete-count]')!.textContent).toBe('2');

    const stage = document.querySelector<HTMLSelectElement>('#org-project-field-stage')!;
    stage.value = 'development';
    stage.dispatchEvent(new Event('change', {bubbles: true}));
    expect(summary.querySelector('[data-summary-control="org-project-field-stage"] [data-summary-value]')!.textContent).toBe('Development');
  });

  test('ignores pages without a marked create form', () => {
    document.body.innerHTML = '<form></form>';
    expect(() => initOrgProjectCreateSummary()).not.toThrow();
  });
});

describe('organization project repository search', {concurrent: false}, () => {
  beforeEach(() => {
    document.body.innerHTML = `
      <section class="org-project-repository-panel">
        <div data-org-project-repository-id="7"></div>
        <form>
          <div class="ui search" data-org-project-repository-search data-owner-id="3" data-select-error="Select a repository">
            <input class="prompt" required>
            <div class="results"></div>
          </div>
          <input type="hidden" name="repository_id">
        </form>
      </section>
    `;
    vi.mocked(GET).mockReset();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test('selects a visible unlinked repository and invalidates edited selections', async () => {
    vi.mocked(GET).mockResolvedValue({
      ok: true,
      json: async () => ({
        data: [
          {repository: {id: 7, full_name: 'org3/linked'}},
          {repository: {id: 8, full_name: 'org3/repo8'}},
        ],
      }),
    } as Response);
    initOrgProjectRepositorySearch();

    const form = document.querySelector<HTMLFormElement>('form')!;
    const input = form.querySelector<HTMLInputElement>('.prompt')!;
    const repositoryID = form.querySelector<HTMLInputElement>('[name="repository_id"]')!;
    input.value = 'repo';
    input.dispatchEvent(new Event('input', {bubbles: true}));
    await vi.advanceTimersByTimeAsync(200);

    expect(GET).toHaveBeenCalledWith('/repo/search?q=repo&uid=3', expect.objectContaining({signal: expect.any(AbortSignal)}));
    expect(document.querySelectorAll('.result')).toHaveLength(1);
    expect(document.querySelector('.result')!.textContent).toContain('org3/repo8');

    document.querySelector<HTMLElement>('.result')!.dispatchEvent(new MouseEvent('mousedown', {bubbles: true}));
    expect(input.value).toBe('org3/repo8');
    expect(repositoryID.value).toBe('8');

    input.value = 'changed';
    input.dispatchEvent(new Event('input', {bubbles: true}));
    expect(repositoryID.value).toBe('');

    const submit = new SubmitEvent('submit', {bubbles: true, cancelable: true});
    expect(form.dispatchEvent(submit)).toBe(false);
    expect(input.validationMessage).toBe('Select a repository');
  });
});
