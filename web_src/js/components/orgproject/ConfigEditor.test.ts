import {createApp, nextTick} from 'vue';
import ConfigEditor from './ConfigEditor.vue';
import {normalizeSchema, validateSchemaClient} from './types.ts';
import type {OrgProjectSchema} from './types.ts';

const labels: Record<string, string> = {addField: 'Add field', newField: 'Field {n}'};

function schema(fields: OrgProjectSchema['fields']): OrgProjectSchema {
  return {schema_version: 1, fields, list_view: {columns: []}, filters: [], metrics: []};
}

describe('ConfigEditor schema helpers', () => {
  test('mounts reactive schema and serializes normalized config', async () => {
    const el = document.createElement('div');
    expect(() => createApp(ConfigEditor, {schema: schema([]), labels}).mount(el)).not.toThrow();
    await nextTick();

    const textarea = el.querySelector<HTMLTextAreaElement>('textarea[name="schema"]')!;
    expect(JSON.parse(textarea.value)).toEqual({...schema([]), list_view: {columns: [], sort: []}});

    const addFieldButton = Array.from(el.querySelectorAll('button')).find((button) => button.textContent === 'Add field')!;
    addFieldButton.click();
    await nextTick();

    expect(JSON.parse(textarea.value).fields).toEqual([
      {key: 'field_1', label: 'Field 1', type: 'short_text', order: 0},
    ]);
  });

  test('renders indexed configuration sections and checkbox column choices', async () => {
    const el = document.createElement('div');
    createApp(ConfigEditor, {schema: schema([
      {key: 'stage', label: '阶段', type: 'single_select', order: 0},
      {key: 'owner', label: '负责人', type: 'member', order: 1},
    ]), labels}).mount(el);
    await nextTick();

    expect(el.querySelector('#org-project-settings-fields')).not.toBeNull();
    expect(el.querySelector('#org-project-settings-list-view')).not.toBeNull();
    expect(el.querySelector('#org-project-settings-filters')).not.toBeNull();
    expect(el.querySelector('#org-project-settings-metrics')).not.toBeNull();
    expect(el.querySelector('select[multiple]')).toBeNull();
    expect(el.querySelector<HTMLInputElement>('input[type="checkbox"][value="stage"]')).not.toBeNull();
    expect(el.querySelector<HTMLInputElement>('input[type="checkbox"][value="owner"]')).not.toBeNull();
  });

  test('normalizes stable field and option ordering', () => {
    const result = normalizeSchema(schema([{key: 'stage', label: 'Stage', type: 'single_select', order: 40, options: [
      {key: 'done', label: 'Done', order: 20},
      {key: 'planned', label: 'Planned', order: 10},
    ]}]));

    expect(result.fields[0].order).toBe(0);
    expect(result.fields[0].options!.map((option) => option.order)).toEqual([0, 1]);
  });

  test('reports duplicate and malformed keys before submission', () => {
    const errors = validateSchemaClient(schema([
      {key: 'Bad Key', label: '', type: 'short_text', order: 0},
      {key: 'Bad Key', label: 'Duplicate', type: 'short_text', order: 1},
    ]));

    expect(errors).toContain('Invalid field key: Bad Key');
    expect(errors).toContain('Duplicate field key: Bad Key');
    expect(errors).toContain('Field Bad Key needs a label');
  });

  test('uses unified dropdowns throughout the configuration editor', async () => {
    const el = document.createElement('div');
    const configuredSchema: OrgProjectSchema = {
      ...schema([
        {key: 'stage', label: 'Stage', type: 'single_select', order: 0},
        {key: 'owner', label: 'Owner', type: 'member', order: 1},
      ]),
      list_view: {columns: ['stage'], sort: [{field_key: 'stage', direction: 'asc'}]},
      filters: [{key: 'risk', label: 'Risk', field_key: 'stage', operator: 'eq'}],
      metrics: [{key: 'projects', label: 'Projects', aggregation: 'count', field_key: 'stage', group_by: 'owner'}],
    };
    createApp(ConfigEditor, {schema: configuredSchema, labels}).mount(el);
    await nextTick();

    const dropdowns = Array.from(el.querySelectorAll('select'));
    expect(dropdowns.length).toBeGreaterThan(1);
    for (const dropdown of dropdowns) {
      expect(Array.from(dropdown.classList)).toEqual(expect.arrayContaining(['ui', 'fluid', 'selection', 'dropdown', 'org-project-dropdown']));
    }
  });
});
