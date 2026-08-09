import {initOrgProjectCreateSummary} from './org-project.ts';

describe('organization project create summary', () => {
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
