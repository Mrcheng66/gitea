import {createApp, nextTick} from 'vue';
import ProjectHistory from './ProjectHistory.vue';

const locale = {
  changed: 'changed', user: 'User', source: 'Source', sourceWeb: 'Web', sourceApi: 'API',
  sourceLegacyImport: 'Legacy import', details: 'View change details', before: 'Before', after: 'After', requestID: 'Request ID',
};

describe('ProjectHistory', () => {
  test('renders readable metadata and changed values', async () => {
    const changes = [{
      id: 11,
      actor_id: 2,
      actor_name: 'User Two',
      actor_link: '/user2',
      request_id: 'request-11',
      changed_fields: ['name', 'values.stage', 'values.labels'],
      before: {
        name: 'Old project',
        values: [
          {key: 'stage', text: 'planning'},
          {key: 'labels', json: '["backend"]'},
        ],
      },
      after: {
        name: 'New project',
        values: [
          {key: 'stage', text: 'development'},
          {key: 'labels', json: '["backend","ui"]'},
        ],
      },
      source: 'web',
      created_at: '2026-08-11T06:30:00Z',
    }];
    const el = document.createElement('div');
    createApp(ProjectHistory, {
      changes,
      fieldLabels: {name: 'Name', 'values.stage': 'Stage', 'values.labels': 'Labels'},
      emptyText: 'No history',
      locale,
    }).mount(el);
    await nextTick();

    expect(el.querySelector<HTMLAnchorElement>('a[href="/user2"]')?.textContent).toBe('User Two');
    expect(el.textContent).toContain('User Two');
    expect(el.textContent).toContain('changed');
    expect(el.textContent).toContain('Stage');
    expect(el.textContent).toContain('Source: Web');
    expect(el.textContent).toContain('request-11');

    const values = Array.from(el.querySelectorAll('pre'), (node) => node.textContent);
    expect(values).toContain('Old project');
    expect(values).toContain('New project');
    expect(values).toContain('planning');
    expect(values).toContain('development');
    expect(values).toContain('[\n  "backend"\n]');
    expect(values).toContain('[\n  "backend",\n  "ui"\n]');
  });

  test('falls back to the actor ID and stable field key', async () => {
    const el = document.createElement('div');
    createApp(ProjectHistory, {
      changes: [{
        id: 12,
        actor_id: 99,
        request_id: 'request-12',
        changed_fields: ['values.removed_field'],
        before: {values: [{key: 'removed_field', text: 'old'}]},
        after: {values: [{key: 'removed_field', text: 'new'}]},
        source: 'legacy-import',
        created_at: '2026-08-11T06:31:00Z',
      }],
      fieldLabels: {},
      emptyText: 'No history',
      locale,
    }).mount(el);
    await nextTick();

    expect(el.textContent).toContain('User #99');
    expect(el.textContent).toContain('removed_field');
    expect(el.textContent).toContain('Source: Legacy import');
  });
});
