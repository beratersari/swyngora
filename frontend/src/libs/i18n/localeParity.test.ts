import { describe, expect, it } from 'vitest';
import { resources } from './resources';

function deepKeys(obj: unknown, prefix = ''): string[] {
  if (obj === null || typeof obj !== 'object' || Array.isArray(obj)) {
    return prefix ? [prefix] : [];
  }
  return Object.entries(obj as Record<string, unknown>).flatMap(([k, v]) => {
    const path = prefix ? `${prefix}.${k}` : k;
    if (v !== null && typeof v === 'object' && !Array.isArray(v)) {
      return deepKeys(v, path);
    }
    return [path];
  });
}

describe('locale key parity', () => {
  it('en and tr share the same key trees for every namespace', () => {
    const en = resources.en;
    const tr = resources.tr;
    const namespaces = Object.keys(en) as (keyof typeof en)[];
    for (const ns of namespaces) {
      const enKeys = deepKeys(en[ns]).sort();
      const trKeys = deepKeys(tr[ns]).sort();
      expect(trKeys, `namespace ${ns}`).toEqual(enKeys);
    }
  });
});
