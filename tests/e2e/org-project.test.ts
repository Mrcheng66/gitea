import {test, expect} from '@playwright/test';
import {apiCreateOrg, apiDeleteOrg, login, randomString} from './utils.ts';

test('publish configuration and create a native organization project', async ({page, request}) => {
  const orgName = `e2e-native-project-${randomString(8)}`;
  const projectSlug = `project-${randomString(8)}`;
  const projectName = 'Native project acceptance';

  await Promise.all([apiCreateOrg(request, orgName), login(page)]);

  try {
    await page.goto(`/org/${orgName}/settings/projects`);
    await page.getByRole('button', {name: 'Publish draft'}).click();
    await expect(page.getByText('The project configuration has been published.')).toBeVisible();

    await page.goto(`/org/${orgName}/projects/new`);
    await expect(page.getByRole('heading', {name: 'Basic information'})).toBeVisible();
    await expect(page.getByRole('heading', {name: 'Plan and responsibility'})).toBeVisible();
    await expect(page.getByRole('heading', {name: 'Status and action'})).toBeVisible();
    await expect(page.getByRole('heading', {name: 'Creation checklist'})).toBeVisible();

    for (const width of [320, 375, 414, 768]) {
      await page.setViewportSize({width, height: 800});
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    }
    await page.setViewportSize({width: 1280, height: 800});

    await page.getByLabel('Name').fill(projectName);
    await page.getByLabel('Slug').fill(projectSlug);
    await page.getByLabel('Description').fill('Created by the native organization project Playwright flow.');
    await expect(page.locator('[data-summary-control="slug"] [data-summary-value]')).toHaveText(projectSlug);
    await expect(page.locator('[data-summary-control="org-project-field-stage"] [data-summary-value]')).toHaveText('规划中');
    await page.getByRole('button', {name: 'Create project'}).click();

    await expect(page).toHaveURL(new RegExp(`/org/${orgName}/projects/${projectSlug}$`));
    await expect(page.getByRole('heading', {name: projectName})).toBeVisible();
    await expect(page.getByRole('link', {name: 'Activity'})).toBeVisible();
    await expect(page.getByRole('link', {name: 'History'})).toBeVisible();

    await page.goto(`/org/${orgName}/projects/list`);
    await expect(page.getByText(projectName)).toBeVisible();
  } finally {
    await apiDeleteOrg(request, orgName);
  }
});
