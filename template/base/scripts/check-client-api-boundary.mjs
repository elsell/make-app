#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import ts from '../apps/web/node_modules/typescript/lib/typescript.js';

const root = path.resolve(process.argv[2] ?? '.');
const sourceRoots = ['apps/web/src', 'apps/mobile/app', 'apps/mobile/src'];
const extensions = new Set(['.ts', '.tsx', '.js', '.jsx', '.svelte']);
const approvedExternalImports = new Set([
  '@sveltejs/kit', 'expo-auth-session', 'expo-constants', 'expo-crypto',
  'expo-localization', 'expo-router', 'expo-secure-store', 'expo-web-browser',
  'node:assert/strict', 'node:test', 'oidc-client-ts', 'react', 'react-native',
  'svelte',
]);
for (const packagePath of [
  'packages/api-client/package.json',
  'packages/client-core/package.json',
  'packages/i18n/package.json',
]) {
  try {
    const manifest = JSON.parse(fs.readFileSync(path.join(root, packagePath), 'utf8'));
    if (typeof manifest.name === 'string' && manifest.name) approvedExternalImports.add(manifest.name);
  } catch { /* Missing or malformed manifests cannot expand the import allowlist. */ }
}
const providerImports = new Map([
  ['apps/mobile/app/index.tsx', new Set(['expo-auth-session', 'expo-web-browser'])],
  ['apps/web/src/lib/auth.ts', new Set(['oidc-client-ts'])],
]);
const allProviderImports = new Set([...providerImports.values()].flatMap((values) => [...values]));
const safeGlobals = new Set([
  'Date', 'Error', 'JSON', 'Math', 'Number', 'Object', 'Promise', 'Set', 'URL',
  'clearTimeout', 'crypto', 'setTimeout',
]);
const safeWindowMembers = new Set(['location', 'sessionStorage']);

function filesBelow(directory) {
  if (!fs.existsSync(directory)) return [];
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const target = path.join(directory, entry.name);
    if (entry.isDirectory()) {
      return ['node_modules', 'build', 'dist', '.svelte-kit'].includes(entry.name) ? [] : filesBelow(target);
    }
    return extensions.has(path.extname(entry.name)) ? [target] : [];
  });
}

function scriptsIn(file, source) {
  if (!file.endsWith('.svelte')) return [source];
  return [...source.matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/gi)].map((match) => match[1]);
}

function declarationNames(sourceFile) {
  const names = new Set();
  function addBinding(name) {
    if (ts.isIdentifier(name)) names.add(name.text);
    else if (ts.isObjectBindingPattern(name) || ts.isArrayBindingPattern(name)) {
      for (const element of name.elements) if (ts.isBindingElement(element)) addBinding(element.name);
    }
  }
  function visit(node) {
    if (ts.isImportClause(node)) {
      if (node.name) names.add(node.name.text);
      if (node.namedBindings) {
        if (ts.isNamespaceImport(node.namedBindings)) names.add(node.namedBindings.name.text);
        else for (const element of node.namedBindings.elements) names.add(element.name.text);
      }
    }
    if (ts.isVariableDeclaration(node) || ts.isParameter(node)) addBinding(node.name);
    if ((ts.isFunctionDeclaration(node) || ts.isClassDeclaration(node)) && node.name) names.add(node.name.text);
    ts.forEachChild(node, visit);
  }
  visit(sourceFile);
  return names;
}

function rootIdentifier(expression) {
  let current = expression;
  while (ts.isPropertyAccessExpression(current) || ts.isElementAccessExpression(current)) current = current.expression;
  while (ts.isParenthesizedExpression(current) || ts.isAsExpression(current) || ts.isNonNullExpression(current)) current = current.expression;
  return ts.isIdentifier(current) ? current.text : undefined;
}

function firstMember(expression, globalName) {
  let current = expression;
  while (ts.isPropertyAccessExpression(current) || ts.isElementAccessExpression(current)) {
    const member = ts.isPropertyAccessExpression(current)
      ? current.name.text
      : ts.isStringLiteral(current.argumentExpression) ? current.argumentExpression.text : undefined;
    if (ts.isIdentifier(current.expression) && current.expression.text === globalName) return member;
    current = current.expression;
  }
  return undefined;
}

function importAllowed(specifier, relative) {
  if (specifier.startsWith('.') || specifier.startsWith('$')) return true;
  if (!approvedExternalImports.has(specifier)) return false;
  if (!allProviderImports.has(specifier)) return true;
  return providerImports.get(relative)?.has(specifier) ?? false;
}

function inspectSource(relative, source, index) {
  const kind = relative.endsWith('.tsx') || relative.endsWith('.jsx') ? ts.ScriptKind.TSX : ts.ScriptKind.TS;
  const sourceFile = ts.createSourceFile(`${relative}#${index}`, source, ts.ScriptTarget.Latest, true, kind);
  if (sourceFile.parseDiagnostics.length > 0) return true;
  const declared = declarationNames(sourceFile);
  let violation = false;
  function visit(node) {
    if (ts.isIdentifier(node) && ['globalThis', 'navigator', 'self'].includes(node.text)) violation = true;
    if ((ts.isPropertyAccessExpression(node) || ts.isElementAccessExpression(node)) && rootIdentifier(node) === 'window') {
      const member = firstMember(node, 'window');
      if (!member || !safeWindowMembers.has(member)) violation = true;
    }
    if (ts.isImportDeclaration(node) && ts.isStringLiteral(node.moduleSpecifier) && !importAllowed(node.moduleSpecifier.text, relative)) {
      violation = true;
    }
    if (ts.isCallExpression(node)) {
      if (node.expression.kind === ts.SyntaxKind.ImportKeyword) violation = true;
      const rootName = rootIdentifier(node.expression);
      if (rootName === 'globalThis' || rootName === 'navigator' || rootName === 'self') violation = true;
      else if (rootName && rootName !== 'window' && !declared.has(rootName) && !safeGlobals.has(rootName)) violation = true;
    }
    if (ts.isNewExpression(node)) {
      const rootName = rootIdentifier(node.expression);
      if (rootName && !declared.has(rootName) && !safeGlobals.has(rootName)) violation = true;
    }
    ts.forEachChild(node, visit);
  }
  visit(sourceFile);
  return violation;
}

const violations = [];
for (const file of sourceRoots.flatMap((sourceRoot) => filesBelow(path.join(root, sourceRoot)))) {
  const relative = path.relative(root, file).split(path.sep).join('/');
  const source = fs.readFileSync(file, 'utf8');
  if (scriptsIn(file, source).some((script, index) => inspectSource(relative, script, index))) violations.push(relative);
}
for (const relative of [...new Set(violations)].sort()) {
  console.error(`client API boundary: ${relative} uses client transport outside an approved generated or provider adapter`);
}
process.exitCode = violations.length > 0 ? 1 : 0;
