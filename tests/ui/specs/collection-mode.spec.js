// UI tests for the Dynamic collection toggle on the plugin settings page (#280).
// global-setup.js renders page.html without a config (manual intervals) and
// page-dynamic.html with COLLECTION_MODE="dynamic" and saved non-default
// intervals; every network call is mocked here.
const { test, expect } = require('@playwright/test');
const fs = require('node:fs');
const path = require('node:path');

const BASE = 'http://uma.test';
const RENDER = path.join(__dirname, '..', '.render');
const JQUERY = path.join(__dirname, '..', 'node_modules', 'jquery', 'dist', 'jquery.min.js');

// Saved intervals in the page-dynamic.html config (global-setup.js).
const SAVED = { INTERVAL_SYSTEM: '30', INTERVAL_DOCKER: '10', INTERVAL_MOVER: '120' };

const STATUS = {
  mode: 'dynamic',
  collectors: [
    { name: 'docker', status: 'running', reason: 'event', event_driven: true, effective_interval_seconds: 300 },
    { name: 'vm', status: 'running', reason: 'active', event_driven: true, effective_interval_seconds: 30 },
    { name: 'ups', status: 'running', reason: 'idle', event_driven: false, effective_interval_seconds: 300 },
    { name: 'system', status: 'running', reason: 'floor', event_driven: false, effective_interval_seconds: 15 },
    { name: 'docker_update', status: 'running', reason: 'idle', effective_interval_seconds: 21600 },
    { name: 'zfs', status: 'stopped', reason: 'idle', hardware_gated: true, effective_interval_seconds: 0 },
    { name: 'mover', status: 'stopped', reason: 'manual', effective_interval_seconds: 0 },
  ],
};

const CAPABILITIES = {
  checked_at: '2026-10-18T09:00:00Z',
  items: [
    { name: 'hardware.ups', state: 'present', detail: 'apcupsd' },
    { name: 'hardware.zfs', state: 'absent', detail: 'no pools' },
  ],
};

const json = (body, status = 200) => ({ status, contentType: 'application/json', body: JSON.stringify(body) });

// Opens a rendered page. status() answers each collectors_status request with
// a route.fulfill() option object. Returns the posted diagnostics requests.
async function openPage(page, html, status = () => json(STATUS)) {
  const requests = [];
  await page.route('**/*', async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname === '/') {
      return route.fulfill({ contentType: 'text/html', body: fs.readFileSync(path.join(RENDER, html), 'utf8') });
    }
    if (url.pathname === '/jquery.js') {
      return route.fulfill({ contentType: 'application/javascript', body: fs.readFileSync(JQUERY, 'utf8') });
    }
    if (url.pathname.endsWith('/include/diagnostics.php')) {
      const form = Object.fromEntries(new URLSearchParams(route.request().postData() || ''));
      requests.push(form);
      if (form.action === 'capabilities') return route.fulfill(json(CAPABILITIES));
      if (form.action === 'collectors_status') return route.fulfill(status());
      return route.fulfill(json({ error: 'unexpected' }, 400));
    }
    if (url.pathname.endsWith('/include/exec.php')) {
      return route.fulfill(json({ running: true }));
    }
    return route.fulfill({ status: 404, body: '' });
  });
  await page.goto(`${BASE}/`);
  return requests;
}

const toggle = (page) => page.locator('label:has(#collection_mode)');
const grid = (page) => page.locator('#uma-interval-grid');
const summary = (page) => page.locator('#uma-collector-summary');
const summaryRow = (page, name) => page.locator(`.uma-summary-row[data-uma-collector="${name}"]`);
const formFields = (page) =>
  page.locator('#collection_mode').evaluate((el) => Object.fromEntries(new FormData(el.closest('form'))));

test('the toggle switches between the interval grid and the summary', async ({ page }) => {
  await openPage(page, 'page.html');
  await expect(page.locator('#collection_mode')).not.toBeChecked();
  await expect(page.locator('#collection_mode_val')).toHaveValue('manual');
  await expect(grid(page)).toBeVisible();
  await expect(summary(page)).toBeHidden();

  await toggle(page).click();
  await expect(page.locator('#collection_mode')).toBeChecked();
  await expect(page.locator('#collection_mode_val')).toHaveValue('dynamic');
  await expect(grid(page)).toBeHidden();
  await expect(summary(page)).toBeVisible();

  await toggle(page).click();
  await expect(page.locator('#collection_mode_val')).toHaveValue('manual');
  await expect(grid(page)).toBeVisible();
  await expect(summary(page)).toBeHidden();
});

test('hidden intervals are still submitted and shown again after switching back', async ({ page }) => {
  await openPage(page, 'page-dynamic.html');
  await expect(page.locator('#collection_mode')).toBeChecked();
  await expect(grid(page)).toBeHidden();

  const fields = await formFields(page);
  expect(fields.COLLECTION_MODE).toBe('dynamic');
  expect(fields.COLLECTION_MODE_CB).toBe('on');
  for (const [name, value] of Object.entries(SAVED)) expect(fields[name], name).toBe(value);
  const selects = await page.locator('.uma-interval-select').evaluateAll((els) => els.map((s) => [s.name, s.disabled]));
  expect(selects.length).toBeGreaterThan(20);
  for (const [name, disabled] of selects) {
    expect(disabled, name).toBe(false);
    expect(fields[name], name).toBeDefined();
  }

  await toggle(page).click();
  await expect(grid(page)).toBeVisible();
  for (const [name, value] of Object.entries(SAVED)) {
    await expect(page.locator(`select[name="${name}"]`), name).toHaveValue(value);
  }
  expect((await formFields(page)).COLLECTION_MODE).toBe('manual');
});

test('the summary renders what each collector does from the status response', async ({ page }) => {
  const requests = await openPage(page, 'page-dynamic.html');
  await page.evaluate(() => updateServiceStatus(true));
  await expect.poll(() => requests.filter((r) => r.action === 'collectors_status').length).toBeGreaterThan(0);

  await expect(summaryRow(page, 'docker')).toHaveText('Docker — on change (events) · 300 s safety poll');
  await expect(summaryRow(page, 'vm')).toHaveText('Virtual Machines — on change (events) · 30 s while a client uses the data');
  await expect(summaryRow(page, 'ups')).toHaveText('UPS Status — 300 s (idle)');
  await expect(summaryRow(page, 'system')).toHaveText('System Metrics — 15 s (required by alert rules)');
  await expect(summaryRow(page, 'docker_update')).toHaveText('Docker Update Checks — 6 h (idle)');
  await expect(summaryRow(page, 'mover')).toHaveText('Mover Status — off');
  await expect(summary(page)).not.toContainText('still uses the manual intervals');

  // Absent hardware follows the grid's visibility: hidden unless Show all collectors.
  await expect(summaryRow(page, 'zfs')).toBeHidden();
  await page.locator('label:has(#show_all_collectors)').click();
  await expect(summaryRow(page, 'zfs')).toBeVisible();
  await expect(summaryRow(page, 'zfs')).toHaveText('ZFS Pools — not detected');
  await expect(summaryRow(page, 'zfs')).toHaveAttribute('title', /^Not detected: /);
});

test('the summary says when the agent still runs on manual intervals', async ({ page }) => {
  await openPage(page, 'page.html', () =>
    json({ mode: 'manual', collectors: [{ name: 'docker', status: 'running', reason: 'manual', effective_interval_seconds: 30 }] }),
  );
  await page.evaluate(() => updateServiceStatus(true));
  await toggle(page).click();
  await expect(summaryRow(page, 'docker')).toHaveText('Docker — every 30 s');
  await expect(summary(page)).toContainText('Click Apply to switch to dynamic collection');
});

test('Re-check hardware works in both modes', async ({ page }) => {
  const requests = await openPage(page, 'page-dynamic.html');
  await page.evaluate(() => updateServiceStatus(true));
  const btn = page.locator('#uma-hw-recheck-btn');
  const rechecks = () => requests.filter((r) => r.action === 'capabilities' && r.refresh === '1').length;

  for (const mode of ['dynamic', 'manual']) {
    await expect(page.locator('#collection_mode_val')).toHaveValue(mode);
    await expect(btn, mode).toBeVisible();
    await expect(btn, mode).toBeEnabled();
    const before = rechecks();
    await btn.click();
    await expect.poll(rechecks, mode).toBe(before + 1);
    await expect(btn, mode).toHaveValue('Re-check hardware');
    await expect(page.locator('#uma-hw-status'), mode).toContainText('Detected hardware: UPS');
    if (mode === 'dynamic') await toggle(page).click();
  }
});

test('shows a fallback while the service is stopped or unreachable', async ({ page }) => {
  let up = true;
  await openPage(page, 'page-dynamic.html', () => (up ? json(STATUS) : json({ error: 'not running' }, 502)));
  // The rendered page has no running agent (no pidof).
  await expect(summary(page)).toContainText('The service is not running');
  await expect(page.locator('.uma-summary-row')).toHaveCount(0);

  await page.evaluate(() => updateServiceStatus(true));
  await expect(summaryRow(page, 'docker')).toBeVisible();

  up = false;
  await page.evaluate(() => loadCollectorStatus());
  await expect(summary(page)).toContainText('Collector status unavailable');
  await expect(page.locator('.uma-summary-row')).toHaveCount(0);

  await page.evaluate(() => updateServiceStatus(false));
  await expect(summary(page)).toContainText('The service is not running');
});
