import {createApp, nextTick} from 'vue';
import ProjectDashboard from './ProjectDashboard.vue';

describe('ProjectDashboard', () => {
  test('renders risk summary values and filter links', async () => {
    const el = document.createElement('div');
    createApp(ProjectDashboard, {
      summary: {active: 8, blocked: 2, overdue: 1, due_soon: 3, average_progress: 62.4},
      baseLink: '/org/rd/projects',
      labels: {
        blocked: 'Blocked', overdue: 'Overdue', dueSoon: 'Due soon', averageProgress: 'Average progress',
        blockedDescription: 'Act now', overdueDescription: 'Past target', dueSoonDescription: 'Confirm delivery', averageProgressDescription: '8 active',
      },
    }).mount(el);
    await nextTick();

    expect(el.textContent).toContain('62%');
    expect(el.querySelector<HTMLAnchorElement>('a[href="/org/rd/projects?filter_risk=blocked"]')).not.toBeNull();
    expect(el.querySelector<HTMLAnchorElement>('a[href="/org/rd/projects?due=overdue"]')).not.toBeNull();
  });
});
