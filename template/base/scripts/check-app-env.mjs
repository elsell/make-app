import { readFile, readdir } from 'node:fs/promises';
import { extname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const toolOwned = new Map([
  ['MAKE_APP_UID', 'Compose host ownership'],
  ['MAKE_APP_GID', 'Compose host ownership'],
  ['GODEBUG', 'Go runtime settings'],
  ['CGO_ENABLED', 'Go compiler settings'],
  ['HOST', 'SvelteKit adapter-node listener'],
  ['NODE_ENV', 'Node runtime mode'],
  ['NPM_CONFIG_PREFIX', 'npm installation path'],
  ['PATH', 'operating-system executable search path'],
  ['PORT', 'SvelteKit adapter-node listener'],
  ['POSTGRES_DB', 'PostgreSQL image contract'],
  ['POSTGRES_PASSWORD', 'PostgreSQL image contract'],
  ['POSTGRES_USER', 'PostgreSQL image contract'],
]);

function discoveredNames(contents, path) {
  const names = new Set();
  const computedProcess = [];
  const computedGo = [];
  const lineNumber = (index) => contents.slice(0, index).split('\n').length;
  const javascriptSource = /\.(?:cjs|js|jsx|mjs|svelte|ts|tsx)$/.test(path);
  if (javascriptSource) {
    for (const pattern of [/\bprocess\.env\.([A-Z][A-Z0-9_]*)/g, /\benv\.([A-Z][A-Z0-9_]*)/g]) {
      for (const match of contents.matchAll(pattern)) names.add(match[1]);
    }
    for (const match of contents.matchAll(/\b(?:process\.env|env)\s*(?:\?\.)?\s*\[([^\]]*)\]/g)) {
      const literal = /^\s*(['"])([A-Z][A-Z0-9_]*)\1\s*$/.exec(match[1]);
      if (literal) names.add(literal[2]);
      else computedProcess.push(lineNumber(match.index));
    }
    for (const match of contents.matchAll(/\b(?:const|let|var)\s*\{([^}]*)\}\s*=\s*process\.env\b/g)) {
      for (const entry of match[1].split(',')) {
        if (entry.trim().startsWith('...')) { computedProcess.push(lineNumber(match.index)); continue; }
        const name = /^\s*([A-Z][A-Z0-9_]*)\b/.exec(entry)?.[1];
        if (name) names.add(name);
      }
    }
  }
  if (/Dockerfile(?:\.[^/]*)?$/.test(path)) {
    for (const line of contents.matchAll(/^ENV\s+(.+)$/gm)) {
      for (const assignment of line[1].matchAll(/(?:^|\s)([A-Z][A-Z0-9_]*)=/g)) names.add(assignment[1]);
    }
  }
  if (/\.ya?ml$/.test(path)) {
    for (const match of contents.matchAll(/["']?([A-Z][A-Z0-9_]*)["']?\s*:/g)) names.add(match[1]);
    for (const match of contents.matchAll(/^\s*-\s*([A-Z][A-Z0-9_]*)=/gm)) names.add(match[1]);
  }
  if (/\.env(?:\.|$)/.test(path)) {
    for (const match of contents.matchAll(/^([A-Z][A-Z0-9_]*)=/gm)) names.add(match[1]);
  }
  if (path.endsWith('.json')) {
    try {
      const visit = (value, environment = false) => {
        if (!value || typeof value !== 'object') return;
        for (const [key, child] of Object.entries(value)) {
          if (environment && /^[A-Z][A-Z0-9_]*$/.test(key)) names.add(key);
          visit(child, key === 'env' || key === 'environment');
        }
      };
      visit(JSON.parse(contents));
    } catch { /* JSON validity belongs to configuration parsing. */ }
  }
  if (path.endsWith('.go')) {
    for (const match of contents.matchAll(/\bos\.(?:Getenv|LookupEnv)\s*\(([^)]*)\)/g)) {
      const literal = /^\s*(["`])([A-Z][A-Z0-9_]*)\1\s*$/.exec(match[1]);
      if (literal) names.add(literal[2]);
      else computedGo.push(lineNumber(match.index));
    }
  }
  if (/\.ya?ml$/.test(path)) {
    for (const match of contents.matchAll(/\$\{([A-Z][A-Z0-9_]*)/g)) names.add(match[1]);
  }
  return { names, computedProcess, computedGo };
}

function hasUnreviewedComputedAccess(contents, accessLines) {
  const lines = contents.split('\n');
  const consumedAnnotations = new Set();
  for (const accessLine of accessLines) {
    const annotationIndex = accessLine - 2;
    const annotation = annotationIndex >= 0 ? lines[annotationIndex] : '';
    const reviewed = /^\s*\/\/ __APP_SLUG__-env-inventory: allow-computed \S.+$/.test(annotation) && !consumedAnnotations.has(annotationIndex);
    if (!reviewed) return true;
    consumedAnnotations.add(annotationIndex);
  }
  return false;
}

export function inspectApplicationEnvironmentSources(sources) {
  const errors = [];
  for (const { path, contents } of sources) {
    const discovered = discoveredNames(contents, path);
    for (const name of discovered.names) {
      if (!name.startsWith('__ENV_PREFIX___') && !toolOwned.has(name)) {
        errors.push(`${path}: ${name} is not __ENV_PREFIX__-prefixed`);
      }
    }
    if (hasUnreviewedComputedAccess(contents, discovered.computedProcess)) errors.push(`${path}: computed process.env access requires explicit review`);
    if (hasUnreviewedComputedAccess(contents, discovered.computedGo)) errors.push(`${path}: computed os environment access requires explicit review`);
  }
  return errors.sort();
}

const applicationSourceExtensions = new Set(['.cjs', '.go', '.js', '.jsx', '.json', '.mjs', '.svelte', '.ts', '.tsx']);
const reviewedVendoredAssets = [
  'apps/api/internal/adapters/httpserver/assets/scalar-api-reference-1.44.20.js',
];

export function isApplicationEnvironmentSourcePath(path) {
  if (reviewedVendoredAssets.some((asset) => path === asset || path.endsWith(`/${asset}`))) return false;
  return applicationSourceExtensions.has(extname(path)) || /(?:^|\/)Dockerfile(?:\.[^/]*)?$|\.ya?ml$|(?:^|\/)\.env(?:\.|$)/.test(path);
}

async function applicationSources(root) {
  const sources = [];
  async function walk(directory) {
    for (const entry of await readdir(directory, { withFileTypes: true })) {
      if (['node_modules', '.svelte-kit', 'build', 'dist', 'android', 'ios'].includes(entry.name)) continue;
      const path = join(directory, entry.name);
      if (entry.isDirectory()) await walk(path);
      else if (!entry.name.includes('.test.') && !entry.name.endsWith('_test.go') && isApplicationEnvironmentSourcePath(path)) {
        sources.push({ path: relative(root, path), contents: await readFile(path, 'utf8') });
      }
    }
  }
  await walk(join(root, 'apps'));
  await walk(join(root, 'packages'));
  for (const path of ['compose.yaml', 'apps/web/Dockerfile', 'scripts/validate-mobile-release-env.mjs']) {
    sources.push({ path, contents: await readFile(join(root, path), 'utf8') });
  }
  sources.push({ path: '.env.example', contents: await readFile(join(root, '.env.example'), 'utf8') });
  return sources;
}

const invoked = process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1];
if (invoked) {
  const root = fileURLToPath(new URL('..', import.meta.url));
  const errors = inspectApplicationEnvironmentSources(await applicationSources(root));
  for (const error of errors) console.error(`application environment check: ${error}`);
  if (errors.length) process.exitCode = 1;
}
