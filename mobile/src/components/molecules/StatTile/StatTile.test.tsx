import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { StatTile } from './StatTile';

describe('StatTile', () => {
  it('renders label and value', () => {
    render(<StatTile label="Open" value="66,000" />);
    expect(screen.getByText('Open')).toBeTruthy();
    expect(screen.getByText('66,000')).toBeTruthy();
  });

  it('shows skeleton when loading', () => {
    const { container } = render(<StatTile label="Open" value="1" isLoading />);
    expect(screen.getByText('Open')).toBeTruthy();
    expect(screen.queryByText('1')).toBeNull();
    expect(container.querySelector('[class]') || container.firstChild).toBeTruthy();
  });
});
