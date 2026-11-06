import { cpSync, mkdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
const source = new URL('../node_modules/pdfjs-dist/', import.meta.url);
const target = new URL('../public/pdfjs/', import.meta.url);
mkdirSync(target, { recursive: true });
for (const name of ['cmaps', 'standard_fonts', 'wasm', 'LICENSE']) {
  cpSync(fileURLToPath(new URL(name, source)), fileURLToPath(new URL(name, target)), { recursive: true });
}
