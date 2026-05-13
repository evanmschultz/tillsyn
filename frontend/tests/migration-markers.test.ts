import { describe, it, expect } from 'vitest';
import * as fs from 'node:fs';
import * as path from 'node:path';

// Resolve paths relative to this test file's location (frontend/tests/).
const frontendDir = path.resolve(__dirname, '..');
const componentsDir = path.join(frontendDir, 'src', 'components');
const vimDir = path.join(frontendDir, 'src', 'lib', 'vim');

const COMPONENT_MARKER = '// MIGRATION TARGET: @hylla/stil-solid';
const VIM_MARKER = '// MIGRATION TARGET: github.com/hylla-org/ro-vim';

/**
 * Recursively collect all files matching the given extensions under a directory.
 * Returns an empty array if the directory does not exist (empty dirs pass vacuously).
 */
function collectFiles(dir: string, extensions: string[]): string[] {
  if (!fs.existsSync(dir)) {
    return [];
  }
  const results: string[] = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      results.push(...collectFiles(full, extensions));
    } else if (extensions.some((ext) => entry.name.endsWith(ext))) {
      results.push(full);
    }
  }
  return results;
}

describe('Migration markers', () => {
  describe('src/components/ — MIGRATION TARGET: @hylla/stil-solid', () => {
    const files = collectFiles(componentsDir, ['.tsx', '.ts']);

    if (files.length === 0) {
      it.skip('no component files yet — passes vacuously with --passWithNoTests', () => {
        // Dirs are empty at D3; assertions accumulate as D4–D9 add files.
      });
    } else {
      for (const file of files) {
        it(`${path.relative(frontendDir, file)} contains migration marker`, () => {
          const content = fs.readFileSync(file, 'utf-8');
          expect(content).toContain(COMPONENT_MARKER);
        });
      }
    }
  });

  describe('src/lib/vim/ — MIGRATION TARGET: github.com/hylla-org/ro-vim', () => {
    const files = collectFiles(vimDir, ['.ts']);

    if (files.length === 0) {
      it.skip('no vim engine files yet — passes vacuously with --passWithNoTests', () => {
        // Dirs are empty at D3; assertions accumulate as D9 adds files.
      });
    } else {
      for (const file of files) {
        it(`${path.relative(frontendDir, file)} contains migration marker`, () => {
          const content = fs.readFileSync(file, 'utf-8');
          expect(content).toContain(VIM_MARKER);
        });
      }
    }
  });
});
