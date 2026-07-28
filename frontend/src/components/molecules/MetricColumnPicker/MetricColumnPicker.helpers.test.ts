import { describe, expect, it } from 'vitest';
import { moveIdByDelta, moveItemAtIndex, toggleIdInList } from './MetricColumnPicker.helpers';

describe('MetricColumnPicker.helpers', () => {
  it('moveItemAtIndex reorders', () => {
    expect(moveItemAtIndex(['a', 'b', 'c'], 0, 2)).toEqual(['b', 'c', 'a']);
    expect(moveItemAtIndex(['a', 'b', 'c'], 2, 0)).toEqual(['c', 'a', 'b']);
  });

  it('moveItemAtIndex is bounds-safe', () => {
    expect(moveItemAtIndex(['a', 'b'], -1, 0)).toEqual(['a', 'b']);
    expect(moveItemAtIndex(['a', 'b'], 0, 9)).toEqual(['a', 'b']);
  });

  it('moveIdByDelta steps by one', () => {
    expect(moveIdByDelta(['a', 'b', 'c'], 'b', -1)).toEqual(['b', 'a', 'c']);
    expect(moveIdByDelta(['a', 'b', 'c'], 'b', 1)).toEqual(['a', 'c', 'b']);
    expect(moveIdByDelta(['a', 'b', 'c'], 'a', -1)).toEqual(['a', 'b', 'c']);
  });

  it('toggleIdInList adds and removes with minCount', () => {
    expect(toggleIdInList(['a'], 'b')).toEqual(['a', 'b']);
    expect(toggleIdInList(['a', 'b'], 'a')).toEqual(['b']);
    expect(toggleIdInList(['a'], 'a')).toEqual(['a']);
  });
});
