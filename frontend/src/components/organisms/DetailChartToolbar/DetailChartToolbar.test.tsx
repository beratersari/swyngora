import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { DetailChartToolbar } from './DetailChartToolbar';

describe('DetailChartToolbar', () => {
  it('renders refresh and calls onRefresh', async () => {
    const user = userEvent.setup();
    const onRefresh = vi.fn();
    renderWithTheme(
      <DetailChartToolbar
        intervals={['1h', '4h']}
        interval="1h"
        onIntervalChange={() => undefined}
        onRefresh={onRefresh}
      />,
    );
    await user.click(screen.getByRole('button', { name: /refresh/i }));
    expect(onRefresh).toHaveBeenCalled();
  });

  it('shows updating tag when isFetching', () => {
    renderWithTheme(
      <DetailChartToolbar
        intervals={['1h']}
        interval="1h"
        onIntervalChange={() => undefined}
        onRefresh={() => undefined}
        isFetching
      />,
    );
    expect(screen.getByText(/updating/i)).toBeInTheDocument();
  });

  it('exposes a single pump threshold select (not dual inputs)', () => {
    renderWithTheme(
      <DetailChartToolbar
        intervals={['1h']}
        interval="1h"
        onIntervalChange={() => undefined}
        pumpThresholdPct={5}
        onPumpThresholdChange={() => undefined}
        showPumpMarkers
        onShowPumpMarkersChange={() => undefined}
      />,
    );
    expect(screen.getByText(/pump threshold|pump eşiği/i)).toBeInTheDocument();
    // One switch for markers, no spinbutton number field
    expect(screen.queryByRole('spinbutton')).not.toBeInTheDocument();
    expect(screen.getByRole('switch')).toBeInTheDocument();
  });

  it('changes the CVD range', async () => {
    const user = userEvent.setup();
    const onRange = vi.fn();
    renderWithTheme(
      <DetailChartToolbar
        intervals={['1h']}
        interval="1h"
        onIntervalChange={() => undefined}
        onShowCvdChange={() => undefined}
        cvdRange="24h"
        onCvdRangeChange={onRange}
      />,
    );
    await user.click(screen.getByRole('combobox', { name: /cvd range|cvd aralığı/i }));
    await user.click(await screen.findByTitle(/^7d$|^7g$/i));
    expect(onRange.mock.calls[0]?.[0]).toBe('7d');
  });

  it('toggles delta, CVD, and liquidation levels', async () => {
    const user = userEvent.setup();
    const onDelta = vi.fn();
    const onCvd = vi.fn();
    const onLevels = vi.fn();
    renderWithTheme(
      <DetailChartToolbar
        intervals={['1h']}
        interval="1h"
        onIntervalChange={() => undefined}
        onShowDeltaChange={onDelta}
        onShowCvdChange={onCvd}
        onShowLiqLevelsChange={onLevels}
      />,
    );
    await user.click(screen.getByRole('switch', { name: /buy − sell|alış − satış/i }));
    await user.click(screen.getByRole('switch', { name: /^cvd$/i }));
    await user.click(screen.getByRole('switch', { name: /liquidation levels|likidasyon seviyeleri/i }));
    expect(onDelta.mock.calls[0]?.[0]).toBe(false);
    expect(onCvd.mock.calls[0]?.[0]).toBe(false);
    expect(onLevels.mock.calls[0]?.[0]).toBe(false);
  });

  it('toggles book, footprint, and crowd versus top', async () => {
    const user = userEvent.setup();
    const onBook = vi.fn();
    const onFoot = vi.fn();
    const onCrowd = vi.fn();
    renderWithTheme(
      <DetailChartToolbar
        intervals={['1h']}
        interval="1h"
        onIntervalChange={() => undefined}
        onShowBookChange={onBook}
        onShowFootprintChange={onFoot}
        onShowCrowdChange={onCrowd}
      />,
    );
    await user.click(screen.getByRole('switch', { name: /book|emir defteri/i }));
    await user.click(screen.getByRole('switch', { name: /footprint|ayak izi/i }));
    await user.click(screen.getByRole('switch', { name: /crowd vs top|kalabalık ve büyükler/i }));
    expect(onBook.mock.calls[0]?.[0]).toBe(false);
    expect(onFoot.mock.calls[0]?.[0]).toBe(false);
    expect(onCrowd.mock.calls[0]?.[0]).toBe(false);
  });

  it('toggles open interest, funding, and liquidation notional', async () => {
    const user = userEvent.setup();
    const onOi = vi.fn();
    const onFunding = vi.fn();
    const onLiq = vi.fn();
    renderWithTheme(
      <DetailChartToolbar
        intervals={['1h']}
        interval="1h"
        onIntervalChange={() => undefined}
        onShowOiChange={onOi}
        onShowFundingChange={onFunding}
        onShowLiqNotionalChange={onLiq}
      />,
    );
    await user.click(screen.getByRole('switch', { name: /open interest|açık pozisyon/i }));
    await user.click(screen.getByRole('switch', { name: /^funding$|^fonlama$/i }));
    await user.click(screen.getByRole('switch', { name: /^liquidations$|^likidasyonlar$/i }));
    expect(onOi.mock.calls[0]?.[0]).toBe(false);
    expect(onFunding.mock.calls[0]?.[0]).toBe(false);
    expect(onLiq.mock.calls[0]?.[0]).toBe(false);
  });

  it('toggles the liquidation map and keeps the range control', async () => {
    const user = userEvent.setup();
    const onMap = vi.fn();
    const onRange = vi.fn();
    renderWithTheme(
      <DetailChartToolbar
        intervals={['1h']}
        interval="1h"
        onIntervalChange={() => undefined}
        showLiqMap
        onShowLiqMapChange={onMap}
        liqMapRange="24h"
        onLiqMapRangeChange={onRange}
        liqMapVenue="combined"
        onLiqMapVenueChange={() => undefined}
        liqMapSide="totals"
        onLiqMapSideChange={() => undefined}
      />,
    );
    expect(screen.getByRole('combobox', { name: /range|aralık/i })).toBeInTheDocument();
    await user.click(screen.getByRole('switch', { name: /liq map|likidasyon haritası/i }));
    expect(onMap.mock.calls[0]?.[0]).toBe(false);
  });

  it('keeps range and venue when only open levels are on', async () => {
    const user = userEvent.setup();
    const onOpen = vi.fn();
    renderWithTheme(
      <DetailChartToolbar
        intervals={['1h']}
        interval="1h"
        onIntervalChange={() => undefined}
        showLiqMap={false}
        showOpenLevels
        onShowOpenLevelsChange={onOpen}
        liqMapRange="24h"
        onLiqMapRangeChange={() => undefined}
        liqMapVenue="combined"
        onLiqMapVenueChange={() => undefined}
        liqMapSide="totals"
        onLiqMapSideChange={() => undefined}
      />,
    );
    expect(screen.getByRole('combobox', { name: /range|aralık/i })).toBeInTheDocument();
    expect(screen.getByRole('combobox', { name: /venue|borsa/i })).toBeInTheDocument();
    expect(screen.queryByRole('combobox', { name: /^side$|^taraf$/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('combobox', { name: /liquidation source|likidasyon kaynağı/i })).not.toBeInTheDocument();
    await user.click(screen.getByRole('switch', { name: /open levels|açık seviyeler/i }));
    expect(onOpen.mock.calls[0]?.[0]).toBe(false);
  });

  it('switches the liquidation map from the model to measured prints', async () => {
    const user = userEvent.setup();
    const onSource = vi.fn();
    renderWithTheme(
      <DetailChartToolbar
        intervals={['1h']}
        interval="1h"
        onIntervalChange={() => undefined}
        showLiqMap
        onShowLiqMapChange={() => undefined}
        liqMapRange="24h"
        onLiqMapRangeChange={() => undefined}
        liqMapVenue="combined"
        onLiqMapVenueChange={() => undefined}
        liqMapSide="totals"
        onLiqMapSideChange={() => undefined}
        liqMapSource="model"
        onLiqMapSourceChange={onSource}
      />,
    );
    await user.click(screen.getByRole('combobox', { name: /liquidation source|likidasyon kaynağı/i }));
    await user.click(await screen.findByTitle('Prints'));
    expect(onSource.mock.calls[0]?.[0]).toBe('prints');
  });
});
