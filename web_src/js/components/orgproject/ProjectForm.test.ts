import {createApp, nextTick} from 'vue';
import ProjectForm from './ProjectForm.vue';
import {initialProjectValues, serializeProjectValues, sortedActiveFields} from './types.ts';
import type {OrgProjectSchema} from './types.ts';

const schema: OrgProjectSchema = {
  schema_version: 1,
  fields: [
    {key: 'archived', label: 'Archived', type: 'short_text', order: 1, archived: true},
    {key: 'stage', label: 'Stage', type: 'single_select', order: 20, default: 'planned', options: [{key: 'planned', label: 'Planned', order: 0}]},
    {key: 'enabled', label: 'Enabled', type: 'boolean', order: 10},
  ],
  list_view: {columns: []},
  filters: [],
  metrics: [],
};

describe('ProjectForm', () => {
  test('initializes defaults and orders active fields', () => {
    expect(initialProjectValues(schema, {})).toEqual({stage: 'planned', enabled: false});
    expect(sortedActiveFields(schema).map((field) => field.key)).toEqual(['enabled', 'stage']);
  });

  test('omits cleared optional values and converts local datetimes', () => {
    const dateSchema: OrgProjectSchema = {
      ...schema,
      fields: [{key: 'due_at', label: 'Due at', type: 'date_time', order: 0}, {key: 'owner', label: 'Owner', type: 'member', order: 1}],
    };
    const result = JSON.parse(serializeProjectValues(dateSchema, {due_at: '2026-08-05T12:30', owner: null}));
    expect(result.owner).toBeUndefined();
    expect(result.due_at).toMatch(/^2026-08-05T/);
    expect(result.due_at).toMatch(/Z$/);
  });

  test('serializes dynamic values into the native form field', async () => {
    const el = document.createElement('div');
    createApp(ProjectForm, {schema, initialValues: {stage: 'planned'}, members: []}).mount(el);
    await nextTick();

    const checkbox = el.querySelector<HTMLInputElement>('input[type="checkbox"]')!;
    checkbox.checked = true;
    checkbox.dispatchEvent(new Event('change', {bubbles: true}));
    await nextTick();

    const hidden = el.querySelector<HTMLInputElement>('input[name="values"]')!;
    expect(JSON.parse(hidden.value)).toEqual({stage: 'planned', enabled: true});
  });
});
