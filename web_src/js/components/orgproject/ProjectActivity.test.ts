import {createApp, nextTick} from 'vue';
import ProjectActivity from './ProjectActivity.vue';
import type {OrgProjectActivitySummary} from './types.ts';

const locale = {
  openPulls: 'Open pull requests', mergedPulls: 'Merged pull requests', releases: 'Releases', repositories: 'Repositories',
  repository: 'Repository', recentCommits: 'Recent commits', latestRelease: 'Latest release', empty: 'No activity', since: 'Since',
};

describe('ProjectActivity', () => {
  test('renders only the visible repository data supplied by the service', async () => {
    const summary: OrgProjectActivitySummary = {
      since: '2026-07-06T12:00:00Z', open_pulls: 2, merged_pulls: 1, release_count: 1,
      repositories: [{id: 32, full_name: 'org/visible', link: '/org/visible', open_pulls: 2, merged_pulls: 1, release_count: 1}],
      commits: [{
        repository_id: 32, repository_full_name: 'org/visible', repository_link: '/org/visible', sha: 'abcdef123456',
        short_sha: 'abcdef1234', link: '/org/visible/commit/abcdef123456', message: 'Visible commit', author_name: 'Alice',
        committed_at: '2026-08-05T10:00:00Z',
      }],
    };
    const el = document.createElement('div');
    createApp(ProjectActivity, {summary, locale}).mount(el);
    await nextTick();

    expect(el.textContent).toContain('org/visible');
    expect(el.textContent).toContain('Visible commit');
    expect(el.textContent).not.toContain('hidden');
    expect(el.querySelector<HTMLAnchorElement>('a[href="/org/visible/commit/abcdef123456"]')).not.toBeNull();
  });
});
