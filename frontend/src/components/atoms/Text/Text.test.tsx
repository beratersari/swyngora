import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { Text } from './Text';

describe('Text', () => {
  it('renders children with body variant by default', () => {
    renderWithTheme(<Text>Hello markets</Text>);
    expect(screen.getByText('Hello markets')).toBeInTheDocument();
  });

  it('shows skeleton when isLoading', () => {
    const { container } = renderWithTheme(
      <Text isLoading skeletonWidth={120}>
        Hidden
      </Text>,
    );
    expect(screen.queryByText('Hidden')).not.toBeInTheDocument();
    expect(container.querySelector('.ant-skeleton')).toBeTruthy();
  });
});
