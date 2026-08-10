import {test, expect} from '@playwright/test';
import {apiCreateOrg, apiDeleteOrg, login, randomString} from './utils.ts';

test('publish configuration and create a native organization project', async ({page, request}) => {
  const orgName = `e2e-native-project-${randomString(8)}`;
  const projectSlug = `project-${randomString(8)}`;
  const projectName = 'Native project acceptance';

  await Promise.all([apiCreateOrg(request, orgName), login(page)]);

  try {
    await page.goto(`/org/${orgName}/settings/projects`);
    await expect(page.getByRole('heading', {name: 'Project fields'})).toBeVisible();
    await expect(page.getByRole('heading', {name: 'List view'})).toBeVisible();
    await expect(page.getByRole('heading', {name: 'Filters'})).toBeVisible();
    await expect(page.getByRole('heading', {name: 'Metrics'})).toBeVisible();
    await expect(page.locator('.org-project-column-option input[value="stage"]')).toBeAttached();

    for (const width of [320, 375, 414, 768]) {
      await page.setViewportSize({width, height: 800});
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    }
    await page.setViewportSize({width: 1280, height: 800});

    await page.getByRole('button', {name: 'Publish draft'}).click();
    await expect(page.getByText('The project configuration has been published.')).toBeVisible();

    await page.goto(`/org/${orgName}/projects/new`);
    await expect(page.getByRole('heading', {name: 'Basic information'})).toBeVisible();
    await expect(page.getByRole('heading', {name: 'Plan and responsibility'})).toBeVisible();
    await expect(page.getByRole('heading', {name: 'Status and action'})).toBeVisible();
    await expect(page.getByRole('heading', {name: 'Creation checklist'})).toBeVisible();

    const stageSelect = page.locator('#org-project-field-stage');
    const stageDropdown = stageSelect.locator('..');
    await expect(stageDropdown).toHaveClass(/org-project-dropdown/);
    await expect(stageDropdown).toHaveClass(/selection/);

    const ownerSelect = page.locator('#org-project-field-owner');
    const ownerDropdown = ownerSelect.locator('..');
    await expect(ownerDropdown).toHaveClass(/search/);
    await ownerDropdown.click();
    await expect(ownerDropdown.locator('input.search')).toBeVisible();
    await page.keyboard.press('Escape');

    const startDate = page.locator('#org-project-field-start_date');
    await expect(startDate).toHaveClass(/org-project-date-input/);
    await startDate.fill('2026-08-10');
    expect(await startDate.evaluate((element) => element.getBoundingClientRect().height)).toBe(38);

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
    await stageDropdown.click();
    await stageDropdown.locator('.menu > .item[data-value="development"]').click();
    await expect(page.locator('[data-summary-control="org-project-field-stage"] [data-summary-value]')).toHaveText('研发中');
    await page.getByRole('button', {name: 'Create project'}).click();

    await expect(page).toHaveURL(new RegExp(`/org/${orgName}/projects/${projectSlug}$`));
    await expect(page.getByRole('heading', {name: projectName})).toBeVisible();
    await expect(page.getByRole('link', {name: 'Activity'})).toBeVisible();
    await expect(page.getByRole('link', {name: 'History'})).toBeVisible();

    await page.goto(`/org/${orgName}/projects/list`);
    await expect(page.getByText(projectName)).toBeVisible();

    const toolbarPrimary = page.locator('.org-project-toolbar-primary');
    const toolbarSecondary = page.locator('.org-project-toolbar-secondary');
    await expect(toolbarPrimary.locator('#org-project-search')).toBeVisible();
    await expect(toolbarPrimary.locator('#org-project-filter-owner')).toBeAttached();
    await expect(toolbarPrimary.locator('#org-project-filter-risk')).toBeAttached();
    await expect(toolbarPrimary.locator('#org-project-filter-stage')).toBeAttached();
    await expect(toolbarPrimary.locator('#org-project-due')).toBeAttached();
    await expect(toolbarPrimary.locator('label')).toHaveCount(5);
    for (const label of await toolbarPrimary.locator('label').all()) {
      await expect(label).toHaveCSS('position', 'absolute');
      await expect(label).toHaveCSS('width', '1px');
      await expect(label).toHaveCSS('height', '1px');
      await expect(label).toHaveCSS('overflow', 'hidden');
    }
    await expect(toolbarSecondary.locator('input[name="mine"]')).toBeVisible();
    await expect(toolbarSecondary.locator('input[name="include_archived"]')).toBeVisible();
    await expect(toolbarSecondary.getByRole('button', {name: 'Filter'})).toBeVisible();

    const primaryFields = toolbarPrimary.locator('.org-project-toolbar-field');
    await expect(primaryFields).toHaveCount(5);
    const primaryFieldY = await Promise.all((await primaryFields.all()).map(async (field) => (await field.boundingBox())!.y));
    expect(Math.max(...primaryFieldY) - Math.min(...primaryFieldY)).toBeLessThanOrEqual(1);

    const primaryBox = (await toolbarPrimary.boundingBox())!;
    const secondaryBox = (await toolbarSecondary.boundingBox())!;
    expect(secondaryBox.y).toBeGreaterThanOrEqual(primaryBox.y + primaryBox.height);

    for (const width of [320, 375, 414, 768]) {
      await page.setViewportSize({width, height: 800});
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    }
  } finally {
    await apiDeleteOrg(request, orgName);
  }
});
