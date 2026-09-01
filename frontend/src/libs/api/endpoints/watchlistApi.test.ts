import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

describe('watchlistApi', () => {
  it('sends baseVersion on add and remove', () => {
    const src = readFileSync(join(dirname(fileURLToPath(import.meta.url)), 'watchlistApi.ts'), 'utf8');
    expect(src).toMatch(/baseVersion: body\.baseVersion/);
    expect(src).toMatch(/typeof baseVersion === 'number' \? \{ baseVersion \}/);
  });
});
