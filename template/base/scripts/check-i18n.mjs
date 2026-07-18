#!/usr/bin/env node
import { readFile, readdir } from 'node:fs/promises';
import { extname, join, relative, resolve } from 'node:path';

const root = resolve(import.meta.dirname, '..');
const localeDir = join(root, 'packages/i18n/src/locales');
let failed = false;

function report(message) {
  console.error(`i18n check: ${message}`);
  failed = true;
}

function parameters(message) {
  return [...message.matchAll(/{{\s*([\w.-]+)\s*}}/g)].map((match) => match[1]).sort();
}

const localeFiles = (await readdir(localeDir)).filter((file) => file.endsWith('.json')).sort();
if (!localeFiles.includes('en.json') || localeFiles.length < 2) {
  report('English and at least one additional complete locale are required');
}

const catalogs = new Map();
for (const file of localeFiles) {
  try {
    catalogs.set(file, JSON.parse(await readFile(join(localeDir, file), 'utf8')));
  } catch (error) {
    report(`${file} is not valid JSON: ${error.message}`);
  }
}

const reference = catalogs.get('en.json') ?? {};
const referenceKeys = Object.keys(reference).sort();
for (const [file, catalog] of catalogs) {
  const keys = Object.keys(catalog).sort();
  for (const key of referenceKeys.filter((key) => !keys.includes(key))) report(`${file} is missing ${key}`);
  for (const key of keys.filter((key) => !referenceKeys.includes(key))) report(`${file} has unexpected ${key}`);
  for (const key of referenceKeys) {
    if (typeof catalog[key] !== 'string' || catalog[key].trim() === '') report(`${file} has an empty or non-string ${key}`);
    if (parameters(catalog[key] ?? '').join() !== parameters(reference[key] ?? '').join()) {
      report(`${file} interpolation parameters differ for ${key}`);
    }
  }
}

for (const key of referenceKeys.filter((key) => key.endsWith('_one'))) {
  const other = `${key.slice(0, -4)}_other`;
  if (!referenceKeys.includes(other)) report(`plural ${key} requires ${other}`);
}

async function filesUnder(directory) {
  const found = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    if (['node_modules', '.svelte-kit', 'build', 'dist'].includes(entry.name)) continue;
    const path = join(directory, entry.name);
    if (entry.isDirectory()) found.push(...await filesUnder(path));
    else found.push(path);
  }
  return found;
}

function looksLikeCopy(value) {
  const withoutGeneratedTokens = value.replaceAll(/__[A-Z_]+__/g, '');
  return /[A-Za-zÀ-ÿ]{2}/.test(withoutGeneratedTokens.trim());
}

function looksLikeInlineCopy(value) {
  const trimmed = value.replaceAll(/__[A-Z_]+__/g, '').trim();
  if (/^Bearer \$\{[^}]+}$/.test(trimmed)) return false;
  return /\s/.test(trimmed) || /^[A-ZÀ-Þ]/.test(trimmed);
}

for (const directory of [join(root, 'apps/web/src'), join(root, 'apps/mobile')]) {
  for (const file of await filesUnder(directory)) {
    if (!['.svelte', '.tsx', '.jsx', '.ts', '.js'].includes(extname(file))) continue;
    let source = await readFile(file, 'utf8');
    const scripts = extname(file) === '.svelte'
      ? [...source.matchAll(/<script[^>]*>([\s\S]*?)<\/script>/g)].map((match) => match[1]).join('\n')
      : source;
    if (extname(file) === '.svelte') source = source.replaceAll(/<script[\s\S]*?<\/script>/g, '').replaceAll(/<style[\s\S]*?<\/style>/g, '');
    source = source.replaceAll(/<!--([\s\S]*?)-->/g, '');
    const svelte = extname(file) === '.svelte';
    const textPattern = svelte
      ? />([^<>{}]+)</g
      : /<([A-Z][A-Za-z0-9.]*)\b[^>]*>([^<>{}]+)<\/\1>/g;
    for (const match of source.matchAll(textPattern)) {
      const copy = match[svelte ? 1 : 2];
      if (looksLikeCopy(copy)) report(`${relative(root, file)} contains literal UI copy: ${copy.trim()}`);
    }
    for (const match of source.matchAll(/\b(?:alt|aria-label|placeholder|title)\s*=\s*["']([^"']+)["']/g)) {
      if (looksLikeCopy(match[1])) report(`${relative(root, file)} contains literal UI attribute copy: ${match[1]}`);
    }
    for (const match of source.matchAll(/{\s*["'`]([^"'`]+)["'`]\s*}/g)) {
      if (looksLikeCopy(match[1])) report(`${relative(root, file)} contains literal expression copy: ${match[1]}`);
    }
    for (const expressionMatch of source.matchAll(/{[^{}\n"'`]*(["'`])([^"'`]+)\1[^{}\n]*}/g)) {
      const expression = expressionMatch[0];
      for (const literal of expression.matchAll(/(["'`])([^"'`]+)\1/g)) {
        const beforeLiteral = expression.slice(0, literal.index);
        const afterLiteral = expression.slice(literal.index + literal[0].length);
        if (/[{,]\s*$/.test(beforeLiteral) && /^\s*:/.test(afterLiteral)) continue;
        if (looksLikeInlineCopy(literal[2])) report(`${relative(root, file)} contains nested literal expression copy: ${literal[2]}`);
      }
    }
    for (const match of scripts.matchAll(/\b(?:const|let|var)\s+\w+\s*(?::[^=;]+)?=\s*["'`]([^"'`]+)["'`]/gi)) {
      if (looksLikeInlineCopy(match[1])) report(`${relative(root, file)} contains literal copy variable: ${match[1]}`);
    }
  }
}

if (failed) process.exit(1);
console.log(`i18n check passed (${localeFiles.length} complete locales)`);
