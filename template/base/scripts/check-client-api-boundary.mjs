#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { createHash } from 'node:crypto';
import ts from '../apps/web/node_modules/typescript/lib/typescript.js';

const root = path.resolve(process.argv[2] ?? '.');
const sourceRoots = ['apps/web/src', 'apps/mobile/app', 'apps/mobile/src'];
const extensions = new Set(['.ts', '.tsx', '.js', '.jsx', '.mjs', '.cjs', '.svelte']);
const approvedSourceRoots = sourceRoots.map((sourceRoot) => path.join(root, sourceRoot));
const webLibRoot = path.join(root, 'apps/web/src/lib');
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
  ['apps/mobile/src/provider-auth.ts', new Set(['expo-auth-session', 'expo-web-browser'])],
  ['apps/web/src/lib/provider-auth.ts', new Set(['oidc-client-ts'])],
]);
const allProviderImports = new Set([...providerImports.values()].flatMap((values) => [...values]));
const safeGlobals = new Set([
  'Boolean', 'Date', 'Error', 'JSON', 'Math', 'Number', 'Object', 'Promise', 'Set', 'URL',
  'clearTimeout', 'crypto', 'setTimeout',
]);
const safeWindowMembers = new Set(['location', 'sessionStorage']);
const browserGlobalReferences = new Set([
  'content', 'document', 'frameElement', 'frames', 'globalThis',
  'navigator', 'open', 'opener', 'parent', 'self', 'top', 'window',
]);
const windowProxyMembers = new Set(['contentWindow', 'defaultView', 'view']);
const protectedProviderAdapters = new Set([
  'apps/mobile/src/provider-auth.ts',
  'apps/mobile/src/provider-auth-state.ts',
  'apps/web/src/lib/provider-auth.ts',
]);
const protectedProviderManifest = 'scripts/protected-provider-adapters.sha256';
const networkPrimitiveReferences = new Set([
  'EventSource', 'Request', 'RTCPeerConnection', 'SharedWorker', 'WebSocket',
  'WebTransport', 'Worker', 'XMLHttpRequest', 'fetch', 'require', 'sendBeacon',
]);

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

function protectedAdapterViolations() {
  let manifest;
  try { manifest = fs.readFileSync(path.join(root, protectedProviderManifest), 'utf8'); }
  catch { return [protectedProviderManifest]; }
  const entries = new Map();
  for (const line of manifest.trim().split(/\r?\n/)) {
    const match = /^([0-9a-f]{64})  (.+)$/.exec(line);
    if (!match || entries.has(match[2])) return [protectedProviderManifest];
    entries.set(match[2], match[1]);
  }
  if (entries.size !== protectedProviderAdapters.size ||
    [...protectedProviderAdapters].some((adapter) => !entries.has(adapter))) return [protectedProviderManifest];
  const violations = [];
  for (const adapter of protectedProviderAdapters) {
    try {
      const digest = createHash('sha256').update(fs.readFileSync(path.join(root, adapter))).digest('hex');
      if (digest !== entries.get(adapter)) violations.push(adapter);
    } catch { violations.push(adapter); }
  }
  return violations;
}

function scriptsIn(file, source) {
  if (!file.endsWith('.svelte')) return [source];
  return [...source.matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/gi)].map((match) => match[1]);
}

function lexicalBindings(sourceFile) {
  const bindings = new Map();
  function hasModifier(node, kind) {
    return ts.canHaveModifiers(node) && ts.getModifiers(node)?.some((modifier) => modifier.kind === kind);
  }
  function isAmbient(node) {
    if (sourceFile.isDeclarationFile) return true;
    for (let current = node; current; current = current.parent) {
      if (hasModifier(current, ts.SyntaxKind.DeclareKeyword)) return true;
    }
    return false;
  }
  function runtimeEnum(node) {
    return !isAmbient(node) && !hasModifier(node, ts.SyntaxKind.ConstKeyword);
  }
  function addBindingTo(scope, name) {
    const names = bindings.get(scope);
    if (ts.isIdentifier(name)) names.add(name.text);
    else if (ts.isObjectBindingPattern(name) || ts.isArrayBindingPattern(name)) {
      for (const element of name.elements) if (ts.isBindingElement(element)) addBindingTo(scope, element.name);
    }
  }
  function addBinding(name) {
    addBindingTo(scopeStack.at(-1), name);
  }
  function isLexicalScope(node) {
    const classLike = ts.isClassDeclaration(node) || ts.isClassExpression(node);
    return ts.isSourceFile(node) || ts.isFunctionLike(node) || ts.isBlock(node) ||
      ts.isModuleDeclaration(node) || ts.isModuleBlock(node) ||
      ts.isCatchClause(node) || ts.isForStatement(node) || ts.isForInStatement(node) ||
      ts.isForOfStatement(node) || ts.isCaseBlock(node) || classLike;
  }
  let scopeStack = [];
  function visit(node) {
    const outerScope = scopeStack.at(-1);
    const runtimeFunction = ts.isFunctionDeclaration(node) && Boolean(node.body) && !isAmbient(node);
    const runtimeClass = ts.isClassDeclaration(node) && !isAmbient(node);
    const runtimeNamespace = ts.isModuleDeclaration(node) && ts.isIdentifier(node.name) && !isAmbient(node);
    if ((runtimeFunction || runtimeClass || (ts.isEnumDeclaration(node) && runtimeEnum(node)) || runtimeNamespace) && node.name && outerScope) {
      addBindingTo(outerScope, node.name);
    }
    const opensScope = isLexicalScope(node);
    if (opensScope) {
      if (!bindings.has(node)) bindings.set(node, new Set());
      scopeStack.push(node);
      if ((ts.isFunctionLike(node) || ts.isClassDeclaration(node) || ts.isClassExpression(node)) && node.name) addBinding(node.name);
    }
    if (ts.isImportClause(node)) {
      if (!node.isTypeOnly && node.name) addBinding(node.name);
      if (node.namedBindings) {
        if (!node.isTypeOnly && ts.isNamespaceImport(node.namedBindings)) addBinding(node.namedBindings.name);
        else if (ts.isNamedImports(node.namedBindings)) {
          for (const element of node.namedBindings.elements) if (!node.isTypeOnly && !element.isTypeOnly) addBinding(element.name);
        }
      }
    }
    if (ts.isImportEqualsDeclaration(node) && !node.isTypeOnly && !isAmbient(node)) addBinding(node.name);
    if (ts.isParameter(node)) addBinding(node.name);
    if (ts.isVariableDeclaration(node) && !isAmbient(node)) {
      const declarationList = ts.isVariableDeclarationList(node.parent) ? node.parent : undefined;
      const blockScoped = declarationList && (declarationList.flags & ts.NodeFlags.BlockScoped) !== 0;
      const target = blockScoped || ts.isCatchClause(node.parent)
        ? scopeStack.at(-1)
        : [...scopeStack].reverse().find((scope) => ts.isSourceFile(scope) || ts.isFunctionLike(scope) || ts.isModuleBlock(scope));
      if (target) addBindingTo(target, node.name);
    }
    ts.forEachChild(node, visit);
    if (opensScope) scopeStack.pop();
  }
  visit(sourceFile);
  return bindings;
}

function rootIdentifierNode(expression) {
  let current = expression;
  while (ts.isPropertyAccessExpression(current) || ts.isElementAccessExpression(current)) current = current.expression;
  while (ts.isParenthesizedExpression(current) || ts.isAsExpression(current) || ts.isNonNullExpression(current)) current = current.expression;
  return ts.isIdentifier(current) ? current : undefined;
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

function isNonReferenceName(node) {
  const parent = node.parent;
  const declarationName = (
    ts.isMethodDeclaration(parent) || ts.isPropertyDeclaration(parent) ||
    ts.isPropertySignature(parent) || ts.isMethodSignature(parent) ||
    ts.isGetAccessorDeclaration(parent) || ts.isSetAccessorDeclaration(parent) ||
    ts.isEnumMember(parent) || ts.isJsxAttribute(parent)
  ) && parent.name === node;
  const importExportName = (ts.isImportSpecifier(parent) || ts.isExportSpecifier(parent)) && parent.propertyName === node;
  return (ts.isPropertyAccessExpression(parent) && parent.name === node) ||
    (ts.isPropertyAssignment(parent) && parent.name === node) ||
    (ts.isBindingElement(parent) && parent.propertyName === node) ||
    declarationName || importExportName;
}

function isTypeOnlyReference(node) {
  for (let current = node; current; current = current.parent) {
    if (ts.isTypeNode(current) || ts.isInterfaceDeclaration(current) ||
      ts.isTypeAliasDeclaration(current) || ts.isTypeParameterDeclaration(current)) return true;
    if (ts.isImportDeclaration(current)) {
      const clause = current.importClause;
      if (clause?.isTypeOnly) return true;
      if (ts.isImportSpecifier(node.parent) && node.parent.isTypeOnly) return true;
    }
    if (ts.isExportDeclaration(current) && current.isTypeOnly) return true;
    if (ts.isExportSpecifier(node.parent) && node.parent.isTypeOnly) return true;
    if (ts.isStatement(current) || ts.isSourceFile(current)) return false;
  }
  return false;
}

function belowApprovedSourceRoot(file) {
  return approvedSourceRoots.some((sourceRoot) => file === sourceRoot || file.startsWith(`${sourceRoot}${path.sep}`));
}

function belowRoot(file, sourceRoot) {
  return file === sourceRoot || file.startsWith(`${sourceRoot}${path.sep}`);
}

function resolveImportBase(base) {
  const explicitExtension = path.extname(base);
  if (explicitExtension && !extensions.has(explicitExtension)) return undefined;
  const candidates = explicitExtension
    ? [base]
    : [base, ...[...extensions].flatMap((extension) => [`${base}${extension}`, path.join(base, `index${extension}`)])];
  return candidates.find((candidate) => fs.existsSync(candidate) && fs.statSync(candidate).isFile());
}

function resolveRelativeImport(importer, specifier) {
  return resolveImportBase(path.resolve(path.dirname(importer), specifier));
}

function importAllowed(specifier, relative, file) {
  if (specifier === '$env/dynamic/private') return relative === 'apps/web/src/lib/server/config.ts';
  if (specifier === '$lib' || specifier.startsWith('$lib/')) {
    if (!relative.startsWith('apps/web/src/')) return false;
    const suffix = specifier === '$lib' ? '' : specifier.slice('$lib/'.length);
    const imported = resolveImportBase(path.join(webLibRoot, suffix));
    return Boolean(imported && belowRoot(imported, webLibRoot));
  }
  if (specifier.startsWith('.')) {
    if (specifier === './$types') return true;
    const imported = resolveRelativeImport(file, specifier);
    return Boolean(imported && belowApprovedSourceRoot(imported));
  }
  if (!approvedExternalImports.has(specifier)) return false;
  if (!allProviderImports.has(specifier)) return true;
  return providerImports.get(relative)?.has(specifier) ?? false;
}

function inspectSource(relative, file, source, index) {
  const kind = relative.endsWith('.tsx') || relative.endsWith('.jsx') ? ts.ScriptKind.TSX : ts.ScriptKind.TS;
  const scriptName = relative.endsWith('.svelte') ? `${relative}.script-${index}.ts` : relative;
  const sourceFile = ts.createSourceFile(scriptName, source, ts.ScriptTarget.Latest, true, kind);
  if (sourceFile.parseDiagnostics.length > 0) return true;
  const bindings = lexicalBindings(sourceFile);
  function isLexicallyBound(identifier) {
    for (let current = identifier.parent; current; current = current.parent) {
      if (bindings.get(current)?.has(identifier.text)) return true;
    }
    return false;
  }
  let violation = false;
  if (relative === 'apps/mobile/app/index.tsx') {
    const exported = sourceFile.statements.filter((statement) =>
      (ts.canHaveModifiers(statement) && ts.getModifiers(statement)?.some((modifier) => modifier.kind === ts.SyntaxKind.ExportKeyword)) ||
      ts.isExportDeclaration(statement) || ts.isExportAssignment(statement));
    const home = exported.length === 1 ? exported[0] : undefined;
    if (!home || !ts.isFunctionDeclaration(home) || home.name?.text !== 'Home' || home.parameters.length !== 0 ||
      home.asteriskToken || ts.getModifiers(home)?.some((modifier) => modifier.kind === ts.SyntaxKind.AsyncKeyword) ||
      !ts.getModifiers(home)?.some((modifier) => modifier.kind === ts.SyntaxKind.DefaultKeyword)) violation = true;
  }
  function visit(node) {
    if (ts.isIdentifier(node) && browserGlobalReferences.has(node.text) &&
      !isNonReferenceName(node) && !isLexicallyBound(node)) {
      if (node.text !== 'window') violation = true;
      else {
        const parent = node.parent;
        const directMember = (ts.isPropertyAccessExpression(parent) || ts.isElementAccessExpression(parent)) && parent.expression === node
          ? firstMember(parent, 'window')
          : undefined;
        if (!directMember || !safeWindowMembers.has(directMember)) violation = true;
      }
    }
    if (ts.isIdentifier(node) && networkPrimitiveReferences.has(node.text) &&
      !isNonReferenceName(node) && !isTypeOnlyReference(node) && !isLexicallyBound(node)) violation = true;
    if (ts.isPropertyAccessExpression(node) || ts.isElementAccessExpression(node)) {
      const rootNode = rootIdentifierNode(node);
      if (rootNode?.text === 'window' && !isLexicallyBound(rootNode)) {
        const member = firstMember(node, 'window');
        if (!member || !safeWindowMembers.has(member)) violation = true;
      }
    }
    if (ts.isPropertyAccessExpression(node) || ts.isElementAccessExpression(node)) {
      const member = ts.isPropertyAccessExpression(node)
        ? node.name.text
        : ts.isStringLiteral(node.argumentExpression) ? node.argumentExpression.text : undefined;
      if (member && windowProxyMembers.has(member)) violation = true;
    }
    if (ts.isImportDeclaration(node) && ts.isStringLiteral(node.moduleSpecifier) && !importAllowed(node.moduleSpecifier.text, relative, file)) {
      violation = true;
    }
    if (ts.isExportDeclaration(node)) {
      if (node.moduleSpecifier && ts.isStringLiteral(node.moduleSpecifier)) {
        if (allProviderImports.has(node.moduleSpecifier.text) || !importAllowed(node.moduleSpecifier.text, relative, file)) violation = true;
      }
    }
    if (ts.isImportEqualsDeclaration(node) && ts.isExternalModuleReference(node.moduleReference)) {
      const expression = node.moduleReference.expression;
      if (!expression || !ts.isStringLiteral(expression) || !importAllowed(expression.text, relative, file)) violation = true;
    }
    if (ts.isCallExpression(node)) {
      if (node.expression.kind === ts.SyntaxKind.ImportKeyword) violation = true;
      const rootNode = rootIdentifierNode(node.expression);
      const rootName = rootNode?.text;
      if (rootNode && ['globalThis', 'navigator', 'self'].includes(rootName) && !isLexicallyBound(rootNode)) violation = true;
      else if (rootNode && rootName !== 'window' && !isLexicallyBound(rootNode) && !safeGlobals.has(rootName)) violation = true;
    }
    if (ts.isNewExpression(node)) {
      const rootNode = rootIdentifierNode(node.expression);
      const rootName = rootNode?.text;
      if (rootNode && !isLexicallyBound(rootNode) && !safeGlobals.has(rootName)) violation = true;
    }
    ts.forEachChild(node, visit);
  }
  visit(sourceFile);
  return violation;
}

const violations = protectedAdapterViolations();
for (const file of sourceRoots.flatMap((sourceRoot) => filesBelow(path.join(root, sourceRoot)))) {
  const relative = path.relative(root, file).split(path.sep).join('/');
  const source = fs.readFileSync(file, 'utf8');
  if (scriptsIn(file, source).some((script, index) => inspectSource(relative, file, script, index))) violations.push(relative);
}
for (const relative of [...new Set(violations)].sort()) {
  console.error(`client API boundary: ${relative} uses client transport outside an approved generated or provider adapter`);
}
process.exitCode = violations.length > 0 ? 1 : 0;
