import {createApp, nextTick} from 'vue';
import ProjectForm from './ProjectForm.vue';
import {groupProjectFields, initialProjectValues, serializeProjectValues, sortedActiveFields} from './types.ts';
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

  test('groups known fields and keeps unknown fields in schema order', () => {
    const fields = [
      {key: 'custom_second', label: 'Custom second', type: 'short_text' as const, order: 40},
      {key: 'risk', label: 'Risk', type: 'single_select' as const, order: 30},
      {key: 'owner', label: 'Owner', type: 'member' as const, order: 20},
      {key: 'custom_first', label: 'Custom first', type: 'short_text' as const, order: 10},
    ];

    const groups = groupProjectFields(fields);
    expect(groups.plan.map((field) => field.key)).toEqual(['owner']);
    expect(groups.status.map((field) => field.key)).toEqual(['risk']);
    expect(groups.other.map((field) => field.key)).toEqual(['custom_second', 'custom_first']);
  });

  test('renders grouped fields only when grouped layout is requested', async () => {
    const groupedSchema: OrgProjectSchema = {
      ...schema,
      fields: [
        {key: 'owner', label: 'Owner', type: 'member', order: 10},
        {key: 'risk', label: 'Risk', type: 'single_select', order: 20, options: [{key: 'normal', label: 'Normal', order: 0}]},
        {key: 'custom', label: 'Custom', type: 'short_text', order: 30},
      ],
    };
    const labels = {
      plan: {title: 'Plan', description: 'Plan description'},
      status: {title: 'Status', description: 'Status description'},
      other: {title: 'Other', description: 'Other description'},
    };
    const groupedElement = document.createElement('div');
    createApp(ProjectForm, {schema: groupedSchema, initialValues: {}, members: [], layout: 'grouped', labels}).mount(groupedElement);
    await nextTick();

    expect(Array.from(groupedElement.querySelectorAll('[data-project-field-group]'), (element) => element.getAttribute('data-project-field-group'))).toEqual(['plan', 'status', 'other']);
    expect(groupedElement.querySelector('[data-project-field-group="plan"]')!.textContent).toContain('Owner');
    expect(groupedElement.querySelector('[data-project-field-group="status"]')!.textContent).toContain('Risk');
    expect(groupedElement.querySelector('[data-project-field-group="other"]')!.textContent).toContain('Custom');

    const plainElement = document.createElement('div');
    createApp(ProjectForm, {schema: groupedSchema, initialValues: {}, members: []}).mount(plainElement);
    await nextTick();
    expect(plainElement.querySelector('[data-project-field-group]')).toBeNull();
    expect(plainElement.querySelector('.org-project-form-fields')).not.toBeNull();
  });
});
