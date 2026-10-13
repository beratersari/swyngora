import { describe, expect, it } from 'vitest';
import { situationLeaveGroupKey } from './helpers';

describe('situationLeaveGroupKey', () => {
  it('keeps the still and left groups apart', () => {
    expect(situationLeaveGroupKey('priceCvd', 'agrees', 'still')).toBe('priceCvd:agrees:still');
    expect(situationLeaveGroupKey('priceCvd', 'agrees', 'left')).toBe('priceCvd:agrees:left');
  });
});
