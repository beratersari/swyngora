import { describe, expect, it } from 'vitest';
import { numericFlashDirection } from './FlashValue.helpers';

describe('numericFlashDirection', () => {
  it('detects up and down', () => {
    expect(numericFlashDirection(10, 11)).toBe('up');
    expect(numericFlashDirection('10.5', '9.2')).toBe('down');
    expect(numericFlashDirection('1,234.5', '1,240')).toBe('up');
  });

  it('ignores non-numeric or unchanged', () => {
    expect(numericFlashDirection('—', 1)).toBeNull();
    expect(numericFlashDirection(5, 5)).toBeNull();
    expect(numericFlashDirection(undefined, 3)).toBeNull();
  });
});
