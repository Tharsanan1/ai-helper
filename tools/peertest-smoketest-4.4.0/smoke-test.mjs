import fs from 'node:fs/promises';
import path from 'node:path';
import { chromium } from 'playwright';

function parseArgs(argv) {
  const args = {};
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (!arg.startsWith('--')) continue;
    const key = arg.slice(2);
    const next = argv[i + 1];
    if (!next || next.startsWith('--')) {
      args[key] = true;
      continue;
    }
    args[key] = next;
    i += 1;
  }
  return args;
}

function required(args, key, fallback = '') {
  const value = args[key] ?? process.env[key.toUpperCase().replace(/-/g, '_')] ?? fallback;
  if (value === '' || value === undefined || value === null) {
    throw new Error(`missing required argument --${key}`);
  }
  return String(value);
}

function optional(args, key, fallback = '') {
  const value = args[key] ?? process.env[key.toUpperCase().replace(/-/g, '_')] ?? fallback;
  return value === undefined || value === null ? fallback : String(value);
}

function logStep(message) {
  console.log(`\n============================================================`);
  console.log(message);
  console.log(`============================================================`);
}

function slugify(value) {
  return String(value)
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .replace(/-{2,}/g, '-');
}

async function captureScreenshot(page, cfg, label) {
  if (!cfg.screenshotDir) {
    return;
  }

  if (cfg.screenshotDelayMs > 0) {
    await new Promise((resolve) => setTimeout(resolve, cfg.screenshotDelayMs));
  }

  const filename = `${String(cfg.screenshotIndex).padStart(3, '0')}-${slugify(label)}.png`;
  const outputPath = path.join(cfg.screenshotDir, filename);
  cfg.screenshotIndex += 1;
  await page.screenshot({ path: outputPath });
  console.log(`Saved screenshot: ${outputPath}`);
}

async function waitForNonEmptyTextbox(locator, timeoutMs = 90000) {
  const deadline = Date.now() + timeoutMs;
  let lastValue = '';
  while (Date.now() < deadline) {
    try {
      lastValue = (await locator.inputValue()).trim();
      if (lastValue !== '') {
        return lastValue;
      }
    } catch (_) {
      // The control can re-render while the page is updating. Keep polling.
    }
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error(`timed out waiting for a non-empty textbox value after ${timeoutMs}ms`);
}

async function carbonLogin(page, cfg, username, password) {
  logStep(`Carbon login as ${username}`);
  await page.goto(`${cfg.baseUrl}/carbon/admin/login.jsp`, { waitUntil: 'domcontentloaded' });
  await page.getByRole('textbox', { name: 'Username' }).fill(username);
  await page.getByRole('textbox', { name: 'Password' }).fill(password);
  await captureScreenshot(page, cfg, `before-submit-carbon-login-${username}`);
  await page.getByRole('button', { name: 'Sign-in' }).click();
  await page.waitForLoadState('networkidle');
  await captureScreenshot(page, cfg, `after-carbon-login-${username}`);
}

async function carbonLogout(page, baseUrl, tenantDomain = '') {
  const tenantPrefix = tenantDomain ? `/t/${tenantDomain}` : '';
  logStep(`Carbon logout${tenantDomain ? ` from ${tenantDomain}` : ''}`);
  await page.goto(`${baseUrl}${tenantPrefix}/carbon/admin/logout_action.jsp`, { waitUntil: 'domcontentloaded' });
  await page.waitForLoadState('networkidle');
}

async function ensureTenant(page, cfg) {
  logStep(`Ensuring tenant ${cfg.tenantDomain} exists`);
  await page.goto(`${cfg.baseUrl}/carbon/tenant-mgt/view_tenants.jsp?region=region1&item=govern_view_tenants_menu`, { waitUntil: 'domcontentloaded' });
  await page.waitForLoadState('networkidle');
  await captureScreenshot(page, cfg, `tenant-list-${cfg.tenantDomain}`);

  const tenantCell = page.getByRole('cell', { name: cfg.tenantDomain, exact: true });
  if (await tenantCell.count()) {
    console.log(`Tenant ${cfg.tenantDomain} already exists, skipping creation.`);
    return;
  }

  await page.goto(`${cfg.baseUrl}/carbon/tenant-mgt/add_tenant.jsp?region=region1&item=govern_add_tenants_menu`, { waitUntil: 'domcontentloaded' });
  const visibleFields = page.locator('input[type="text"], input[type="password"], input[type="email"]');
  await visibleFields.nth(0).fill(cfg.tenantDomain);
  await visibleFields.nth(1).fill(cfg.tenantAdminFirstName);
  await visibleFields.nth(2).fill(cfg.tenantAdminLastName);
  await visibleFields.nth(3).fill(cfg.tenantAdminUser);
  await visibleFields.nth(4).fill(cfg.tenantAdminPassword);
  await visibleFields.nth(5).fill(cfg.tenantAdminPassword);
  await visibleFields.nth(6).fill(cfg.tenantAdminEmail);
  await captureScreenshot(page, cfg, `before-submit-create-tenant-${cfg.tenantDomain}`);
  await page.getByRole('button', { name: 'Save' }).click();
  await page.waitForLoadState('networkidle');
  await captureScreenshot(page, cfg, `after-create-tenant-${cfg.tenantDomain}`);

  const successTexts = [
    'Tenant has been successfully added',
    'Successfully added the tenant',
    cfg.tenantDomain,
  ];
  for (const text of successTexts) {
    const match = page.getByText(text, { exact: false });
    if (await match.count()) {
      return;
    }
  }
}

async function ensureTenantUser(page, cfg) {
  logStep(`Ensuring tenant user ${cfg.tenantUser} exists in ${cfg.tenantDomain}`);
  await page.goto(`${cfg.baseUrl}/t/${cfg.tenantDomain}/carbon/user/user-mgt.jsp`, { waitUntil: 'domcontentloaded' });
  await page.waitForLoadState('networkidle');

  await page.locator('input[value="*"]').fill(cfg.tenantUser);
  await captureScreenshot(page, cfg, `before-search-tenant-user-${cfg.tenantUser}`);
  await page.getByRole('button', { name: 'Search Users' }).click();
  await page.waitForLoadState('networkidle');
  await captureScreenshot(page, cfg, `after-search-tenant-user-${cfg.tenantUser}`);

  const existingUser = page.getByRole('cell', { name: cfg.tenantUser, exact: true });
  if (await existingUser.count()) {
    console.log(`Tenant user ${cfg.tenantUser} already exists, skipping creation.`);
    return;
  }

  await page.goto(`${cfg.baseUrl}/t/${cfg.tenantDomain}/carbon/user/add-step1.jsp`, { waitUntil: 'domcontentloaded' });
  await page.locator('input[name="username"]').fill(cfg.tenantUser);
  await page.locator('#password').fill(cfg.tenantUserPassword);
  await page.locator('#password-repeat').fill(cfg.tenantUserPassword);
  await captureScreenshot(page, cfg, `before-submit-tenant-user-step1-${cfg.tenantUser}`);
  await page.getByRole('button', { name: 'Next >' }).click();
  await page.waitForLoadState('networkidle');
  await captureScreenshot(page, cfg, `after-tenant-user-step1-${cfg.tenantUser}`);

  for (const roleName of cfg.tenantUserRoles) {
    await page.locator(`input[type="checkbox"][value="${roleName}"]`).check();
  }
  await captureScreenshot(page, cfg, `before-submit-tenant-user-roles-${cfg.tenantUser}`);
  await page.getByRole('button', { name: 'Finish' }).click();
  await page.waitForLoadState('networkidle');
  await captureScreenshot(page, cfg, `after-create-tenant-user-${cfg.tenantUser}`);
}

async function publisherLogin(page, cfg) {
  const username = `${cfg.tenantUser}@${cfg.tenantDomain}`;
  logStep(`Publisher login as ${username}`);
  await page.goto(`${cfg.baseUrl}/publisher/logout`, { waitUntil: 'domcontentloaded' }).catch(() => {});
  await page.goto(`${cfg.baseUrl}/publisher`, { waitUntil: 'domcontentloaded' });
  await page.getByRole('textbox', { name: 'Username' }).fill(username);
  await page.getByRole('textbox', { name: 'Password' }).fill(cfg.tenantUserPassword);
  await captureScreenshot(page, cfg, `before-submit-publisher-login-${cfg.tenantUser}`);
  await page.getByRole('button', { name: 'Continue' }).click();
  await page.waitForURL(/\/publisher\//, { timeout: 60000 });
  await page.waitForLoadState('networkidle');
  await captureScreenshot(page, cfg, `after-publisher-login-${cfg.tenantUser}`);
}

async function createRestApi(page, cfg) {
  const apiName = `${cfg.apiNamePrefix}-${new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)}`;
  const contextValue = apiName.toLowerCase();
  logStep(`Creating Publisher REST API ${apiName}`);

  await page.goto(`${cfg.baseUrl}/publisher/apis/create/rest`, { waitUntil: 'domcontentloaded' });
  await page.waitForURL(/\/publisher\/apis\/create\/rest/, { timeout: 60000 });

  await page.getByRole('textbox', { name: 'Name*' }).fill(apiName);
  await page.getByRole('textbox', { name: 'Context*' }).fill(contextValue);
  await page.getByRole('textbox', { name: 'Version*' }).fill(cfg.apiVersion);
  await page.getByRole('textbox', { name: 'Endpoint' }).fill(cfg.apiEndpoint);

  const createAndPublishButton = page.getByRole('button', { name: 'Create & Publish', exact: true });
  await createAndPublishButton.waitFor({ state: 'visible', timeout: 30000 });
  await captureScreenshot(page, cfg, `before-submit-create-and-publish-api-${apiName}`);
  await createAndPublishButton.click();
  await page.waitForLoadState('networkidle');

  await page.locator('#itest-api-name-version').filter({ hasText: apiName }).waitFor({ timeout: 60000 });
  await captureScreenshot(page, cfg, `after-create-api-${apiName}`);
  console.log(`Created API: ${apiName}`);
  const apiIdMatch = page.url().match(/\/publisher\/apis\/([^/]+)\/overview/);
  return { apiName, apiId: apiIdMatch?.[1] ?? '' };
}

async function openApiInDevPortal(publisherPage, cfg, apiName, apiId = '') {
  logStep(`Opening ${apiName} in Dev Portal`);
  const viewInDevPortalLink = publisherPage
    .getByRole('link', { name: 'View in Dev Portal', exact: true })
    .or(publisherPage.getByRole('link', { name: 'View in devportal', exact: true }))
    .first();

  await viewInDevPortalLink.waitFor({ state: 'visible', timeout: 30000 });
  await captureScreenshot(publisherPage, cfg, `before-open-devportal-${apiName}`);

  const popupPromise = Promise.race([
    publisherPage.context().waitForEvent('page'),
    new Promise((resolve) => setTimeout(() => resolve(null), 5000)),
  ]);
  await viewInDevPortalLink.click();
  let devPortalPage = await popupPromise;
  if (!devPortalPage) {
    devPortalPage = publisherPage;
  }

  await devPortalPage.waitForLoadState('domcontentloaded');
  await devPortalPage.waitForLoadState('networkidle').catch(() => {});

  const signInButton = devPortalPage.getByRole('button', { name: /sign-in/i });
  if (await signInButton.count()) {
    await signInButton.click();
    await devPortalPage.getByRole('textbox', { name: 'Username' }).fill(`${cfg.tenantUser}@${cfg.tenantDomain}`);
    await devPortalPage.getByRole('textbox', { name: 'Password' }).fill(cfg.tenantUserPassword);
    await captureScreenshot(devPortalPage, cfg, `before-submit-devportal-login-${cfg.tenantUser}`);
    await devPortalPage.getByRole('button', { name: 'Continue' }).click();
    await devPortalPage.waitForLoadState('networkidle');
  }

  if (apiId && !devPortalPage.url().includes(apiId)) {
    await devPortalPage.goto(`${cfg.baseUrl}/devportal/apis/${apiId}/overview?tenant=${cfg.tenantDomain}`, { waitUntil: 'domcontentloaded' });
    await devPortalPage.waitForLoadState('networkidle').catch(() => {});
  }

  await devPortalPage.locator('h2, h1').filter({ hasText: apiName }).first().waitFor({ timeout: 60000 });
  await captureScreenshot(devPortalPage, cfg, `after-open-devportal-${apiName}`);
  return devPortalPage;
}

async function openDefaultApplication(devPortalPage, cfg) {
  await devPortalPage.getByRole('link', { name: 'Applications' }).click();
  await devPortalPage.waitForLoadState('networkidle');
  await captureScreenshot(devPortalPage, cfg, `applications-list-${cfg.tenantDomain}`);
  await devPortalPage.getByRole('link', { name: 'DefaultApplication', exact: true }).click();
  await devPortalPage.waitForLoadState('networkidle');
  await devPortalPage.getByRole('heading', { name: 'DefaultApplication' }).waitFor({ timeout: 30000 });
  await captureScreenshot(devPortalPage, cfg, `default-application-overview-${cfg.tenantDomain}`);
}

async function ensureSubscription(devPortalPage, cfg, apiName) {
  logStep(`Ensuring ${apiName} is subscribed in Dev Portal`);
  await openDefaultApplication(devPortalPage, cfg);
  await devPortalPage.getByRole('link', { name: 'Subscriptions' }).click();
  await devPortalPage.waitForLoadState('networkidle');
  await captureScreenshot(devPortalPage, cfg, `application-subscriptions-${apiName}`);

  const existingSubscription = devPortalPage.getByRole('link', { name: apiName, exact: true });
  if (await existingSubscription.count()) {
    console.log(`API ${apiName} is already subscribed to DefaultApplication.`);
    return;
  }

  await devPortalPage.getByRole('button', { name: 'Subscribe APIs' }).click();
  const subscribeDialog = devPortalPage.getByRole('dialog', { name: 'Subscribe APIs' });
  await subscribeDialog.waitFor({ state: 'visible', timeout: 30000 });
  await subscribeDialog.getByRole('textbox').fill(apiName);
  await captureScreenshot(devPortalPage, cfg, `before-submit-subscribe-api-${apiName}`);
  await subscribeDialog.getByRole('button', { name: 'Subscribe', exact: true }).click();
  await devPortalPage.getByText('Subscription successful', { exact: false }).waitFor({ timeout: 30000 });
  await captureScreenshot(devPortalPage, cfg, `after-subscribe-api-${apiName}`);

  const closeButton = subscribeDialog.getByRole('button').filter({ hasText: /^close$/i });
  if (await closeButton.count()) {
    await closeButton.click().catch(() => {});
  } else {
    await devPortalPage.keyboard.press('Escape').catch(() => {});
  }
}

async function ensureProductionKeys(devPortalPage, cfg) {
  logStep('Ensuring production keys exist in Dev Portal');
  await openDefaultApplication(devPortalPage, cfg);
  await devPortalPage.getByRole('link', { name: 'Production Keys' }).click();
  await devPortalPage.waitForLoadState('networkidle');
  await captureScreenshot(devPortalPage, cfg, `production-keys-page-${cfg.tenantDomain}`);

  const consumerKey = devPortalPage.getByRole('textbox', { name: 'Consumer Key' });
  if (await consumerKey.count()) {
    console.log('Production keys already exist, skipping generation.');
    return;
  }

  await captureScreenshot(devPortalPage, cfg, `before-submit-generate-production-keys-${cfg.tenantDomain}`);
  await devPortalPage.getByRole('button', { name: 'Generate Keys' }).click();
  await devPortalPage.getByText('Application keys generated successfully', { exact: false }).waitFor({ timeout: 60000 });
  await consumerKey.waitFor({ state: 'visible', timeout: 30000 });
  await captureScreenshot(devPortalPage, cfg, `after-generate-production-keys-${cfg.tenantDomain}`);
}

async function tryOutApi(devPortalPage, cfg, apiName, apiId = '') {
  logStep(`Trying out ${apiName} in API Console`);
  await devPortalPage.getByRole('link', { name: 'APIs' }).click();
  await devPortalPage.waitForLoadState('networkidle');

  const apiCard = devPortalPage.getByRole('link', { name: apiName, exact: true }).first();
  if (await apiCard.count()) {
    await apiCard.click();
    await devPortalPage.waitForLoadState('networkidle');
  } else if (apiId) {
    await devPortalPage.goto(`${cfg.baseUrl}/devportal/apis/${apiId}/overview?tenant=${cfg.tenantDomain}`, { waitUntil: 'domcontentloaded' });
    await devPortalPage.waitForLoadState('networkidle');
  } else {
    throw new Error(`could not locate API ${apiName} in Dev Portal`);
  }

  await devPortalPage.getByRole('heading', { name: 'Overview' }).waitFor({ timeout: 30000 });

  const apiConsoleLink = devPortalPage.getByRole('link', { name: 'API Console', exact: true });
  await apiConsoleLink.waitFor({ state: 'visible', timeout: 30000 });
  await apiConsoleLink.click();
  await devPortalPage.waitForLoadState('networkidle');
  await devPortalPage.getByRole('heading', { name: 'API Console' }).waitFor({ timeout: 30000 });
  await captureScreenshot(devPortalPage, cfg, `api-console-${apiName}`);

  const accessToken = devPortalPage.getByRole('textbox', { name: 'Access Token' });
  if ((await accessToken.inputValue()).trim() === '') {
    await captureScreenshot(devPortalPage, cfg, `before-submit-get-test-key-${apiName}`);
    await devPortalPage.getByRole('button', { name: 'GET TEST KEY' }).click();
    await waitForNonEmptyTextbox(accessToken);
    await captureScreenshot(devPortalPage, cfg, `after-get-test-key-${apiName}`);
  }

  await devPortalPage.getByRole('button', { name: 'GET /*', exact: true }).click();
  await devPortalPage.getByRole('button', { name: 'Try it out' }).click();
  await captureScreenshot(devPortalPage, cfg, `before-submit-api-console-execute-${apiName}`);
  await devPortalPage.getByRole('button', { name: 'Execute' }).click();
  await devPortalPage.getByRole('cell', { name: '200', exact: true }).waitFor({ timeout: 60000 });
  await devPortalPage.getByRole('heading', { name: 'Response body' }).waitFor({ timeout: 30000 });
  await captureScreenshot(devPortalPage, cfg, `after-api-console-execute-${apiName}`);

  const responseBody = await devPortalPage.getByRole('code').last().textContent();
  if (!responseBody || !responseBody.includes('"method": "GET"')) {
    throw new Error('API console response did not contain the expected GET payload');
  }
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const cfg = {
    baseUrl: optional(args, 'base-url', 'https://localhost:9443'),
    adminUser: required(args, 'admin-user', 'admin'),
    adminPassword: required(args, 'admin-password', 'admin'),
    tenantDomain: required(args, 'tenant-domain', 'peertest.com'),
    tenantAdminUser: required(args, 'tenant-admin-user', 'peer'),
    tenantAdminPassword: required(args, 'tenant-admin-password', 'peer1'),
    tenantAdminEmail: required(args, 'tenant-admin-email', 'peer@peertest.com'),
    tenantAdminFirstName: optional(args, 'tenant-admin-first-name', 'peer'),
    tenantAdminLastName: optional(args, 'tenant-admin-last-name', 'admin'),
    tenantUser: required(args, 'tenant-user', 'peertestuser'),
    tenantUserPassword: required(args, 'tenant-user-password', 'peer1'),
    tenantUserRoles: ['Internal/creator', 'Internal/publisher', 'Internal/subscriber'],
    apiEndpoint: optional(args, 'api-endpoint', 'https://httpbin.org/anything'),
    apiNamePrefix: optional(args, 'api-name-prefix', 'PeerTestAPI'),
    apiVersion: optional(args, 'api-version', '1.0.0'),
    screenshotDir: optional(args, 'screenshot-dir', ''),
    screenshotDelayMs: Number(optional(args, 'screenshot-delay-ms', '1000')),
    screenshotIndex: 1,
    headless: Boolean(args.headless),
    slowMo: Number(optional(args, 'slow-mo', '250')),
    keepOpen: Boolean(args['keep-open']),
  };

  if (cfg.screenshotDir) {
    cfg.screenshotDir = path.resolve(cfg.screenshotDir);
    await fs.mkdir(cfg.screenshotDir, { recursive: true });
    console.log(`Saving screenshots to: ${cfg.screenshotDir}`);
  }

  const browser = await chromium.launch({
    headless: cfg.headless,
    slowMo: cfg.slowMo,
  });

  const context = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await context.newPage();

  try {
    await carbonLogin(page, cfg, cfg.adminUser, cfg.adminPassword);
    await ensureTenant(page, cfg);
    await carbonLogout(page, cfg.baseUrl);

    await carbonLogin(page, cfg, `${cfg.tenantAdminUser}@${cfg.tenantDomain}`, cfg.tenantAdminPassword);
    await ensureTenantUser(page, cfg);
    await carbonLogout(page, cfg.baseUrl, cfg.tenantDomain);

    await publisherLogin(page, cfg);
    const { apiName, apiId } = await createRestApi(page, cfg);
    const devPortalPage = await openApiInDevPortal(page, cfg, apiName, apiId);
    await ensureSubscription(devPortalPage, cfg, apiName);
    await ensureProductionKeys(devPortalPage, cfg);
    await tryOutApi(devPortalPage, cfg, apiName, apiId);

    logStep('Smoke test completed successfully');
    if (cfg.keepOpen) {
      console.log('Keeping browser open. Press Ctrl+C when finished inspecting.');
      await new Promise(() => {});
    }
  } finally {
    if (!cfg.keepOpen) {
      await browser.close();
    }
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
