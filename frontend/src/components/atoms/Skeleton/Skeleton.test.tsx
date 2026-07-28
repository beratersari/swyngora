import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { Skeleton } from './Skeleton';

describe('Skeleton', () => {
  it('renders chart skeleton with status role', () => {
    renderWithTheme(<Skeleton variant="chart" />);
    expect(screen.getByRole('status')).toBeInTheDocument();
  });

  it('renders children when not loading in wrapper mode', () => {
    renderWithTheme(
      <Skeleton isLoading={false}>
        <span>Content</span>
      </Skeleton>,
    );
    expect(screen.getByText('Content')).toBeInTheDocument();
  });

  it('hides children when loading in wrapper mode', () => {
    const { container } = renderWithTheme(
      <Skeleton isLoading variant="text">
        <span>Content</span>
      </Skeleton>,
    );
    expect(screen.queryByText('Content')).not.toBeInTheDocument();
    expect(container.querySelector('.ant-skeleton')).toBeTruthy();
  });

  it('renders button variant as ant skeleton element', () => {
    const { container } = renderWithTheme(<Skeleton variant="button" />);
    expect(container.querySelector('.ant-skeleton-button')).toBeTruthy();
  });

  it('renders avatar skeleton', () => {
    const { container } = renderWithTheme(<Skeleton variant="avatar" />);
    expect(container.querySelector('.ant-skeleton-avatar')).toBeTruthy();
  });

  it('renders input and card variants', () => {
    const { container: c1 } = renderWithTheme(<Skeleton variant="input" />);
    expect(c1.querySelector('.ant-skeleton-input')).toBeTruthy();
    const { container: c2 } = renderWithTheme(<Skeleton variant="card" />);
    expect(c2.querySelector('[role="status"]')).toBeTruthy();
  });
});
