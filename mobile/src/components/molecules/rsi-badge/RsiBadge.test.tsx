import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { RsiBadge } from './RsiBadge';

describe('RsiBadge', () => {
  it('renders label', () => {
    render(<RsiBadge label="RSI 55.0" tone="secondary" />);
    expect(screen.getByText('RSI 55.0')).toBeTruthy();
  });

  it('shows unavailable dash', () => {
    render(<RsiBadge label="—" />);
    expect(screen.getByText('—')).toBeTruthy();
  });
});
