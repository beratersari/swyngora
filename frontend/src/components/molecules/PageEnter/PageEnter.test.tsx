import { describe, expect, it } from 'vitest';
import { renderWithTheme } from '@/test/render';
import { PageEnter } from './PageEnter';

describe('PageEnter', () => {
  it('renders children', () => {
    const { getByText } = renderWithTheme(
      <PageEnter motionKey="/markets">
        <p>Desk</p>
      </PageEnter>,
    );
    expect(getByText('Desk')).toBeInTheDocument();
  });
});
