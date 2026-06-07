#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';

const repo = process.cwd();
const workspace = path.resolve(repo, '..');
const failures = [];

function resolveWorkspacePath(envName, fallbackRel) {
  const configured = process.env[envName];
  if (configured) return path.resolve(repo, configured);
  return path.resolve(workspace, fallbackRel);
}

function read(rel) {
  const full = path.resolve(repo, rel);
  if (!fs.existsSync(full)) {
    failures.push(`missing file: ${rel}`);
    return '';
  }
  return fs.readFileSync(full, 'utf8');
}

function expectText(label, text, needles) {
  for (const needle of needles) {
    if (!text.includes(needle)) failures.push(`${label} missing ${needle}`);
  }
}

const serviceTypes = read('internal/modules/studio/service_types.go');
expectText('CreateGenerationJobInput Go DTO', serviceTypes, [
  'InputMode',
  'GenerationStrategy',
  'SourceAssetIDs',
  'SourceAssets',
  'StudioSourceAssetInput',
  'json:"input_mode"',
  'json:"generation_strategy"',
  'json:"source_assets"',
]);

const strategy = read('internal/modules/studio/service_strategy.go');
expectText('Studio generation strategy normalizer', strategy, [
  'STUDIO_SOURCE_ASSETS_LIMIT_EXCEEDED',
  'STUDIO_MULTI_IMAGE_PROVIDER_UNSUPPORTED',
  'ask_for_required_input',
  'comfyui_bridge',
  'studioSourceAssetLimit = 4',
]);

const swagger = JSON.parse(read('docs/openapi/swagger.json') || '{}');
const createJob = swagger.definitions?.['internal_modules_studio.CreateGenerationJobInput']?.properties || {};
for (const field of ['input_mode', 'generation_strategy', 'source_asset_ids', 'source_assets']) {
  if (!createJob[field]) failures.push(`OpenAPI CreateGenerationJobInput missing ${field}`);
}
for (const field of ['input_mode', 'generation_strategy']) {
  const enums = createJob[field]?.enum || [];
  for (const value of ['text_to_image', 'image_to_image', 'multi_image', 'ask_for_required_input']) {
    if (!enums.includes(value)) failures.push(`OpenAPI ${field} enum missing ${value}`);
  }
}
const sourceAssetRef = createJob.source_assets?.items?.$ref || '';
if (!sourceAssetRef.endsWith('/internal_modules_studio.StudioSourceAssetInput')) {
  failures.push(`OpenAPI source_assets must reference StudioSourceAssetInput, got ${sourceAssetRef}`);
}
const sourceAssetProps = swagger.definitions?.['internal_modules_studio.StudioSourceAssetInput']?.properties || {};
for (const field of ['asset_id', 'id', 'role', 'label', 'required', 'metadata']) {
  if (!sourceAssetProps[field]) failures.push(`OpenAPI StudioSourceAssetInput missing ${field}`);
}

const frontendTypesPath = path.resolve(resolveWorkspacePath('MENU_FRONTEND_ROOT', 'menu-frontend'), 'src/types/studio.ts');
if (!fs.existsSync(frontendTypesPath)) {
  failures.push(`missing frontend consumer types: ${frontendTypesPath}`);
} else {
  const frontendTypes = fs.readFileSync(frontendTypesPath, 'utf8');
  expectText('menu-frontend CreateJobRequest', frontendTypes, [
    'interface CreateJobRequest',
    'input_mode?: StudioInputMode',
    'generation_strategy?: string',
    'source_asset_ids: string[]',
    'source_assets?: StudioSourceAssetInput[]',
  ]);
  expectText('menu-frontend StudioSourceAssetInput', frontendTypes, [
    'interface StudioSourceAssetInput',
    'asset_id: string',
    'role: StudioSourceAssetRole',
    'label?: string',
    'required?: boolean',
  ]);
}

if (failures.length) {
  console.error('Menu contract consumer sweep FAIL');
  for (const failure of failures) console.error(`- ${failure}`);
  process.exit(1);
}
console.log(JSON.stringify({ status: 'PASS', checked: ['go_dto', 'openapi', 'frontend_types', 'strategy_fail_closed'] }, null, 2));
