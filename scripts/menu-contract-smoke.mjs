#!/usr/bin/env node
const baseURL = (process.env.MENU_BASE_URL || 'http://127.0.0.1:8196').replace(/\/$/, '');
const failures = [];
let token = process.env.MENU_SMOKE_TOKEN || '';

async function request(path, options = {}) {
  const headers = { ...(options.headers || {}) };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (options.body && !headers['Content-Type']) headers['Content-Type'] = 'application/json';
  const res = await fetch(`${baseURL}${path}`, { ...options, headers });
  const text = await res.text();
  let body = null;
  try { body = text ? JSON.parse(text) : null; } catch { body = { raw: text }; }
  return { res, body, text };
}

function dataOf(envelope) {
  if (!envelope || typeof envelope !== 'object') return envelope;
  if ('data' in envelope) return envelope.data;
  return envelope;
}

function assert(condition, message) {
  if (!condition) failures.push(message);
}

async function checkPublic(path, label, validate) {
  const { res, body } = await request(path);
  assert(res.ok, `${label} expected 2xx got ${res.status}`);
  if (res.ok && validate) validate(dataOf(body), body);
}

await checkPublic('/healthz', 'healthz');
await checkPublic('/api/v1/menu/health', 'menu health');
await checkPublic('/api/v1/menu/template-center/meta', 'template meta', data => {
  assert(data && typeof data === 'object', 'template meta should return object data');
});
let firstTemplateId = process.env.MENU_SMOKE_TEMPLATE_ID || '';
await checkPublic('/api/v1/menu/template-center/catalog', 'template catalog', data => {
  const items = Array.isArray(data) ? data : (data?.items || []);
  assert(Array.isArray(items), 'template catalog should return items array');
  firstTemplateId = firstTemplateId || items[0]?.template_id || items[0]?.id || '';
});
if (firstTemplateId) {
  await checkPublic(`/api/v1/menu/template-center/catalog/${encodeURIComponent(firstTemplateId)}`, 'template detail', data => {
    assert(data?.template_id || data?.id, 'template detail should include template_id/id');
    assert(Array.isArray(data?.input_slots || []), 'template detail should include input_slots array');
  });
}
await checkPublic('/api/v1/menu/commercial/offerings', 'commercial offerings', data => {
  assert(data !== undefined, 'commercial offerings should return data');
});

if (process.env.MENU_SMOKE_AUTH === '1' && !token) {
  const email = process.env.MENU_SMOKE_EMAIL || `menu-smoke-${Date.now()}@example.com`;
  const password = process.env.MENU_SMOKE_PASSWORD || `MenuSmoke${Date.now()}!`;
  const register = await request('/api/v1/menu/auth/register', {
    method: 'POST',
    body: JSON.stringify({ email, password, name: 'Menu Smoke', restaurant_name: 'Menu Smoke Restaurant' }),
  });
  assert(register.res.ok, `register expected 2xx got ${register.res.status}`);
  const auth = dataOf(register.body);
  token = auth?.access_token || auth?.token || '';
  assert(token, 'register should return access token');
}

if (token) {
  const session = await request('/api/v1/menu/auth/session');
  assert(session.res.ok, `session expected 2xx got ${session.res.status}`);
  const sessionData = dataOf(session.body);
  assert(sessionData?.access || sessionData?.user, 'session should include access/user');
  const wallet = await request('/api/v1/menu/user/wallet-summary');
  assert(wallet.res.ok, `wallet summary expected 2xx got ${wallet.res.status}`);
  const quota = await request('/api/v1/menu/user/quota-summary');
  assert(quota.res.ok, `quota summary expected 2xx got ${quota.res.status}`);
}

if (failures.length) {
  console.error('Menu contract smoke FAIL');
  for (const failure of failures) console.error(`- ${failure}`);
  process.exit(1);
}
console.log(JSON.stringify({ status: 'PASS', baseURL, auth: Boolean(token), checked_template_id: firstTemplateId || null }, null, 2));
