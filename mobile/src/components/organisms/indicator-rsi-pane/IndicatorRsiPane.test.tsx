import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { IndicatorRsiPane } from './IndicatorRsiPane';

describe('IndicatorRsiPane', () => {
  it('shows latest RSI label', () => {
    render(
      <IndicatorRsiPane data={[]} latestRsi={55.123} isLoading={false} errorMessage={null} />,
    );
    expect(screen.getByText(/RSI \(14\) · latest 55.12/)).toBeTruthy();
  });

  it('shows error', () => {
    render(
      <IndicatorRsiPane
        data={[]}
        latestRsi={null}
        isLoading={false}
        errorMessage="indicator fail"
      />,
    );
    expect(screen.getByText('indicator fail')).toBeTruthy();
  });

  it('shows skeleton when loading with no data', () => {
    const { container } = render(
      <IndicatorRsiPane data={[]} latestRsi={null} isLoading errorMessage={null} />,
    );
    expect(container.firstChild).toBeTruthy();
  });
});
