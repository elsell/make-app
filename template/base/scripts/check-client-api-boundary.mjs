#!/usr/bin/env node
import { readdir, readFile } from 'node:fs/promises';
import { extname, relative, resolve } from 'node:path';

const root = resolve(process.argv[2] ?? '.');
const clientRoots = ['apps/web/src', 'apps/mobile/app', 'apps/mobile/src'];
const sourceExtensions = new Set(['.js', '.mjs', '.ts', '.tsx', '.svelte']);
let failed = false;

async function sourceFiles(directory) {
  const files = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    if (entry.name === 'node_modules' || entry.name === 'build' || entry.name === 'dist' || entry.name === '.svelte-kit') continue;
    const path = resolve(directory, entry.name);
    if (entry.isDirectory()) files.push(...await sourceFiles(path));
    else if (entry.isFile() && sourceExtensions.has(extname(entry.name))) files.push(path);
  }
  return files;
}

function fetchCalls(source) {
  const calls = [];
  const startPattern = /(?:^|[^A-Za-z0-9_$])(?:(?:globalThis|window)\s*\.\s*)?fetch\s*\(/gm;
  for (const match of source.matchAll(startPattern)) {
    const open = source.indexOf('(', match.index + match[0].lastIndexOf('fetch'));
    let depth = 0;
    let quote = '';
    let escaped = false;
    for (let index = open; index < source.length; index += 1) {
      const character = source[index];
      if (quote) {
        if (escaped) escaped = false;
        else if (character === '\\') escaped = true;
        else if (character === quote) quote = '';
        continue;
      }
      if (character === '"' || character === "'" || character === '`') {
        quote = character;
        continue;
      }
      if (character === '(') depth += 1;
      else if (character === ')' && --depth === 0) {
        calls.push(source.slice(open + 1, index));
        break;
      }
    }
  }
  return calls;
}

for (const clientRoot of clientRoots) {
  let files;
  try { files = await sourceFiles(resolve(root, clientRoot)); }
  catch (error) {
    if (error?.code === 'ENOENT') continue;
    throw error;
  }
  for (const file of files) {
    const source = await readFile(file, 'utf8');
    if (fetchCalls(source).some((call) => /\/v1(?:\/|\b)/.test(call))) {
      console.error(`client API boundary: ${relative(root, file)} issues a raw /v1 fetch; use @*/api-client`);
      failed = true;
    }
  }
}

process.exitCode = failed ? 1 : 0;
