#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';

const repo = process.cwd();
const workspace = path.resolve(repo, '..');
const failures = [];
const warnings = [];

function resolveWorkspacePath(envName, fallbackRel) {
  const configured = process.env[envName];
  if (configured) return path.resolve(repo, configured);
  return path.resolve(workspace, fallbackRel);
}

function read(absOrRel, required = true) {
  const full = path.isAbsolute(absOrRel) ? absOrRel : path.resolve(repo, absOrRel);
  if (!fs.existsSync(full)) {
    (required ? failures : warnings).push(`missing ${required ? 'required' : 'optional'} file: ${full}`);
    return '';
  }
  return fs.readFileSync(full, 'utf8');
}

function requireTerms(label, text, terms) {
  for (const term of terms) {
    if (!text.includes(term)) failures.push(`${label} missing ${term}`);
  }
}

const runbookPath = path.resolve(resolveWorkspacePath('MENU_FRONTEND_ROOT', 'menu-frontend'), 'docs/menu-observability-slo-release-readiness.md');
const runbook = read(runbookPath);
requireTerms('Menu observability/SLO runbook', runbook, [
  'Journey SLI/SLO dashboard spec',
  'Correlation checklist',
  'Prod read-only smoke runbook',
  'Local/dev safe live smoke',
  'Release acceptance report template',
  'request_id',
  'trace_id',
  'charge_session_id',
  'runtime_job_id',
  'quota reserve/release',
]);

const criticalJourneys = read('docs/qa/CRITICAL_JOURNEYS.md');
requireTerms('backend critical journeys', criticalJourneys, [
  'PASS_WITH_NOTES',
  'BLOCKED',
  'Four-slot multi-image generation',
  'STUDIO_SOURCE_ASSETS_LIMIT_EXCEEDED',
  'comfyui_bridge',
]);

const openapiReadme = read('docs/openapi/README.md');
requireTerms('OpenAPI README observability baseline', openapiReadme, [
  'request_id',
  'X-Request-ID',
  'X-Trace-ID',
  'Prometheus',
]);

const backendGuide = read('docs/BACKEND_GUIDE.md');
requireTerms('backend guide observability baseline', backendGuide, [
  'structured JSON logging',
  'request_id',
  'trace_id',
  'Prometheus-standard metrics',
]);

const smokeReportPath = path.resolve(workspace, 'reports/evidence-contract/menu-studio-core-chain/v-menu-safe-contract-smoke.json');
if (fs.existsSync(smokeReportPath)) {
  const smoke = JSON.parse(fs.readFileSync(smokeReportPath, 'utf8'));
  if (!['PASS', 'PASS_WITH_NOTES', 'FAIL'].includes(smoke.status)) failures.push(`safe smoke report status invalid: ${smoke.status}`);
  if (!smoke.real_api_evidence?.status) failures.push('safe smoke report missing real_api_evidence.status');
} else {
  warnings.push(`safe smoke report not present yet: ${smokeReportPath}`);
}

const evidencePath = path.resolve(workspace, 'reports/evidence-contract/menu-studio-core-chain/latest.json');
if (fs.existsSync(evidencePath)) {
  const evidence = JSON.parse(fs.readFileSync(evidencePath, 'utf8'));
  if (!evidence.final_status) failures.push('latest evidence contract missing final_status');
  if (!Array.isArray(evidence.blind_spots)) failures.push('latest evidence contract missing blind_spots array');
} else {
  warnings.push(`latest evidence contract not present yet: ${evidencePath}`);
}

const status = failures.length ? 'FAIL' : (warnings.length ? 'PASS_WITH_NOTES' : 'PASS');
const payload = {
  status,
  checked: [
    runbookPath,
    path.resolve(repo, 'docs/qa/CRITICAL_JOURNEYS.md'),
    path.resolve(repo, 'docs/openapi/README.md'),
    path.resolve(repo, 'docs/BACKEND_GUIDE.md'),
    smokeReportPath,
    evidencePath,
  ],
  failures,
  warnings,
};
console.log(JSON.stringify(payload, null, 2));
process.exit(failures.length ? 1 : 0);
