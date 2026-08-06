import {normalizeSchema, validateSchemaClient} from './types.ts';
import type {OrgProjectSchema} from './types.ts';

function schema(fields: OrgProjectSchema['fields']): OrgProjectSchema {
  return {schema_version: 1, fields, list_view: {columns: []}, filters: [], metrics: []};
}

describe('ConfigEditor schema helpers', () => {
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
});
