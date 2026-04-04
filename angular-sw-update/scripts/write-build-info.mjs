import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const projectRoot = path.resolve(scriptDir, '..');
const outputDir = path.join(projectRoot, 'src', 'app');
const outputFile = path.join(outputDir, 'build-info.ts');
const builtAt = new Date().toISOString();
const buildId = builtAt.replace(/[-:]/g, '').replace(/\.\d{3}Z$/, 'Z');

const content = `export const BUILD_INFO = {
  id: '${buildId}',
  builtAt: '${builtAt}',
} as const;
`;

await mkdir(outputDir, { recursive: true });
await writeFile(outputFile, content, 'utf8');

console.log(`Wrote build info to ${path.relative(projectRoot, outputFile)} (${buildId})`);
