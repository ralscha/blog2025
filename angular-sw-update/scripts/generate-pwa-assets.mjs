import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { generateImages } from 'pwa-asset-generator';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const projectRoot = path.resolve(scriptDir, '..');
const iconSource = path.join(scriptDir, 'pwa-icon.svg');
const outputDir = path.join(projectRoot, 'public', 'icons');
const manifestFile = path.join(projectRoot, 'public', 'manifest.webmanifest');
const indexFile = path.join(projectRoot, 'src', 'index.html');

const result = await generateImages(iconSource, outputDir, {
  background: 'linear-gradient(145deg, #fff7ed 0%, #fdba74 55%, #f97316 100%)',
  opaque: true,
  padding: '8%',
  scrape: false,
  manifest: manifestFile,
  index: indexFile,
  pathOverride: 'icons',
  favicon: true,
  mstile: true,
  type: 'png',
});

console.log(
  `Generated ${result.savedImages.length} PWA assets in ${path.relative(projectRoot, outputDir)}`,
);
