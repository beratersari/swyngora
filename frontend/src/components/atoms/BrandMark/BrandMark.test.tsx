import { describe, expect, it } from 'vitest';
import { renderWithTheme } from '@/test/render';
import { BrandMark } from './BrandMark';

describe('BrandMark', () => {
  it('is decorative by default', () => {
    const { container } = renderWithTheme(<BrandMark />);
    const svg = container.querySelector('svg');
    expect(svg).toHaveAttribute('aria-hidden', 'true');
  });

  it('exposes a title when provided', () => {
    const { container } = renderWithTheme(<BrandMark title="Swyngora" />);
    expect(container.querySelector('svg')).toHaveAttribute('role', 'img');
    expect(container.querySelector('title')).toHaveTextContent('Swyngora');
  });
});
