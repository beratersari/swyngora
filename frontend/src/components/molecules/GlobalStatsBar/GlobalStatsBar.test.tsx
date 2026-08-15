import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { GlobalStatsBar } from './GlobalStatsBar';

describe('GlobalStatsBar', () => {
  it('renders counts and bitcoin last', () => {
    const { getByText } = renderWithProviders(
      <GlobalStatsBar coinCount={421} volumeLabel="$48.2B" btcPrice="$63,034" btcChange="+0.55%" btcUp />,
    );
    expect(getByText('421')).toBeInTheDocument();
    expect(getByText('$48.2B')).toBeInTheDocument();
    expect(getByText('$63,034')).toBeInTheDocument();
    expect(getByText('+0.55%')).toBeInTheDocument();
  });
});
