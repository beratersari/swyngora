import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { Skeleton } from './Skeleton';

describe('Skeleton', () => {
  it('renders chart block placeholder', () => {
    const { container } = renderWithTheme(<Skeleton variant="chart" />);
    expect(container.querySelector('[role="status"]')).toBeTruthy();
  });

  it('wrapper shows children when not loading', () => {
    renderWithTheme(
      <Skeleton isLoading={false}>
        <span>Ready</span>
      </Skeleton>,
    );
    expect(screen.getByText('Ready')).toBeInTheDocument();
  });

  it('wrapper hides children when loading', () => {
    renderWithTheme(
      <Skeleton isLoading variant="text">
        <span>Hidden</span>
      </Skeleton>,
    );
    expect(screen.queryByText('Hidden')).not.toBeInTheDocument();
  });
});
