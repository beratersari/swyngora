import { describe, expect, it } from 'vitest';
import { render } from '@testing-library/react';
import { Skeleton } from './Skeleton';

describe('Skeleton', () => {
  it('renders without crash', () => {
    const { container } = render(<Skeleton height={20} width={100} />);
    expect(container.firstChild).toBeTruthy();
  });
});
