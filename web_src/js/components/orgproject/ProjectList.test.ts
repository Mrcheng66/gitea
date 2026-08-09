import {createApp, nextTick} from 'vue';
import ProjectList from './ProjectList.vue';
import type {OrgProjectListRow} from './types.ts';

describe('ProjectList', () => {
  test('renders progress, risk, owner, followers, and updated date', async () => {
    const rows: OrgProjectListRow[] = [{
      Project: {ID: 1, Slug: 'auth', Name: 'Auth', Description: 'Identity platform', Lifecycle: 'active', UpdatedUnix: 1786204800},
      Fields: [
        {Key: 'owner', Label: 'Owner', Value: 'Alice', Type: 'member', Raw: '', Members: [{id: 2, name: 'alice', full_name: 'Alice'}]},
        {Key: 'followers', Label: 'Followers', Value: 'Bob、Carol', Type: 'member_array', Raw: '', Members: [{id: 3, name: 'bob', full_name: 'Bob'}, {id: 4, name: 'carol', full_name: 'Carol'}]},
        {Key: 'progress', Label: 'Progress', Value: '42%', Type: 'percent', Raw: '', Number: 42},
        {Key: 'risk', Label: 'Risk', Value: 'Blocked', Type: 'single_select', Raw: 'blocked'},
      ],
    }];
    const el = document.createElement('div');
    createApp(ProjectList, {rows, baseLink: '/org/rd/projects', archivedLabel: 'Archived', noDescription: 'None', projectLabel: 'Project', updatedLabel: 'Updated'}).mount(el);
    await nextTick();

    expect(el.textContent).toContain('Alice');
    expect(el.querySelectorAll('.org-project-followers .org-project-avatar')).toHaveLength(2);
    expect(el.querySelector<HTMLElement>('.org-project-progress i')!.style.width).toBe('42%');
    expect(el.querySelector('.is-risk-blocked')).not.toBeNull();
  });
});
