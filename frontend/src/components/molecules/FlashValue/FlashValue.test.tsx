import { describe, expect, it } from 'vitest';
import { renderWithTheme } from '@/test/render';
import { FlashValue } from './FlashValue';

describe('FlashValue', () => {
  it('wraps children and flashes when the value rises', () => {
    const { getByText, rerender, container } = renderWithTheme(
      <FlashValue value={10}>
        <span>10</span>
      </FlashValue>,
    );
    expect(getByText('10')).toBeInTheDocument();
    rerender(
      <FlashValue value={12}>
        <span>12</span>
      </FlashValue>,
    );
    expect(container.querySelector('[data-flash="up"]')).toBeTruthy();
    expect(getByText('12')).toBeInTheDocument();
  });
});
