import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { Text } from './Text';

describe('Text', () => {
  it('renders children', () => {
    renderWithTheme(<Text>Hello</Text>);
    expect(screen.getByText('Hello')).toBeInTheDocument();
  });

  it('hides content when isLoading', () => {
    const { container } = renderWithTheme(<Text isLoading>Hello</Text>);
    expect(screen.queryByText('Hello')).not.toBeInTheDocument();
    expect(container.querySelector('.ant-skeleton')).toBeTruthy();
  });

  it('uses title skeleton for heading variants', () => {
    const { container } = renderWithTheme(
      <Text variant="h2" isLoading>
        Title
      </Text>,
    );
    expect(container.querySelector('.ant-skeleton')).toBeTruthy();
  });

  it('supports as polymorphism and mono', () => {
    renderWithTheme(
      <Text as="strong" mono>
        Mono
      </Text>,
    );
    expect(screen.getByText('Mono').tagName.toLowerCase()).toBe('strong');
  });
});
