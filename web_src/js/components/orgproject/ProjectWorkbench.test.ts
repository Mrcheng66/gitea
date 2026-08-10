import {createApp, nextTick} from 'vue';
import ProjectWorkbench from './ProjectWorkbench.vue';
import type {ProjectWorkbenchResult} from './types.ts';

const labels = {
  title: 'R&D workbench', subtitle: 'Delivery evidence', team: 'Team', mine: 'Mine', attention: 'Attention', clear: 'Clear',
  blocked: 'Blocked', overdue: 'Overdue', dueSoon: 'Due soon', stale: 'Stale', unowned: 'Unowned',
  currentProblem: 'Current problem', nextAction: 'Next action', owner: 'Owner', participants: 'Participants',
  realProgress: 'Real progress', people: 'People', quickLinks: 'Quick links', allActivity: 'All activity', empty: 'Empty',
  configure: 'Configure', noEvidence: 'No evidence', activityUnavailable: 'Unavailable', expand: 'Expand', collapse: 'Collapse',
  release: 'Release', pullMerged: 'Merged', issueClosed: 'Completed', commit: 'Commit', target: 'Target',
  releases: 'releases', merged: 'merged', open: 'open',
};

describe('ProjectWorkbench', () => {
  test('connects project decisions, ownership, and delivery evidence', async () => {
    const owner = {id: 1, name: 'alice', full_name: 'Alice', link: '/alice', owned: 1, participating: 0, projects: ['Platform']};
    const participant = {id: 2, name: 'bob', full_name: 'Bob', link: '/bob', owned: 0, participating: 1, projects: ['Platform']};
    const workbench: ProjectWorkbenchResult = {
      attention: {blocked: 1, overdue: 0, due_soon: 1, stale: 0, unowned: 0}, configured_organizations: 1,
      people: [owner],
      projects: [{
        id: 8, name: 'Platform', description: 'Shared platform', link: '/org/projects/platform', organization: 'Org',
        organization_url: '/org', stage_key: 'testing', stage: 'Testing', risk_key: 'blocked', risk: 'Blocked', progress: 68,
        owner, participants: [participant], current_problem: 'Role mapping is incomplete', next_action: 'Finish gray release verification',
        next_action_owner: owner, next_action_due: '2026-08-12', target_date: '2026-08-16', overdue: false, due_soon: true,
        stale: false, expanded: false, activity_error: false, updated_at: '2026-08-09T00:00:00Z',
        activity: {
          since: '2026-08-02T00:00:00Z', repositories: [], commits: [], open_pulls: 2, merged_pulls: 3, release_count: 1,
          partial: false,
          progress: [{
            kind: 'release', title: 'v1.4 gray release', link: '/org/repo/releases/tag/v1.4', repository_id: 2,
            repository_full_name: 'org/repo', repository_link: '/org/repo', author_name: 'Alice', occurred_at: new Date().toISOString(),
          }, {
            kind: 'pull_merged', title: 'Merge permission cache', link: '/org/repo/pulls/12', repository_id: 2,
            repository_full_name: 'org/repo', repository_link: '/org/repo', author_name: 'Bob', occurred_at: new Date().toISOString(),
          }, {
            kind: 'commit', title: 'Add audit detail', link: '/org/repo/commit/123', repository_id: 2,
            repository_full_name: 'org/repo', repository_link: '/org/repo', author_name: 'Alice', occurred_at: new Date().toISOString(),
          }],
        },
      }],
    };
    const el = document.createElement('div');
    createApp(ProjectWorkbench, {workbench, onlyMine: false, baseLink: '/', labels}).mount(el);
    await nextTick();

    expect(el.textContent).toContain('Role mapping is incomplete');
    expect(el.textContent).toContain('Finish gray release verification');
    expect(el.textContent).toContain('v1.4 gray release');
    expect(el.textContent).toContain('1 releases · 3 merged · 2 open');
    expect(el.querySelector<HTMLAnchorElement>('a[href="/org/repo/releases/tag/v1.4"]')).not.toBeNull();

    const execution = el.querySelector<HTMLElement>('.project-execution')!;
    expect(execution.querySelector(':scope > .project-execution-header > .project-responsibility')).not.toBeNull();
    expect(execution.querySelectorAll('.project-responsibility-row')).toHaveLength(2);
    expect(Array.from(execution.children, (child) => child.className)).toEqual([
      'project-execution-header',
      'project-progress-row',
      'project-evidence',
      'project-decision-row',
    ]);
    expect(execution.querySelectorAll('.project-evidence li')).toHaveLength(1);

    execution.querySelector<HTMLButtonElement>('.project-evidence button')!.click();
    await nextTick();
    expect(execution.querySelectorAll('.project-evidence li')).toHaveLength(3);
  });
});
