import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { RSIHeatmap } from './RSIHeatmap';

const frameBox = {
  width: 960,
  height: 520,
  top: 0,
  left: 0,
  bottom: 520,
  right: 960,
  x: 0,
  y: 0,
  toJSON() {
    return {};
  },
} as DOMRect;

const origClientWidth = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'clientWidth');
const origClientHeight = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'clientHeight');

describe('RSIHeatmap', () => {
  beforeEach(() => {
    vi.spyOn(Element.prototype, 'getBoundingClientRect').mockReturnValue(frameBox);
    Object.defineProperty(HTMLElement.prototype, 'clientWidth', {
      configurable: true,
      get: () => 960,
    });
    Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
      configurable: true,
      get: () => 520,
    });
    class FakeRO {
      cb: ResizeObserverCallback;
      constructor(cb: ResizeObserverCallback) {
        this.cb = cb;
      }
      observe(target: Element) {
        const node = target as HTMLElement;
        this.cb(
          [
            {
              target,
              contentRect: { width: node.clientWidth, height: node.clientHeight },
            } as ResizeObserverEntry,
          ],
          this as unknown as ResizeObserver,
        );
      }
      unobserve() {}
      disconnect() {}
    }
    vi.stubGlobal('ResizeObserver', FakeRO);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
    if (origClientWidth) Object.defineProperty(HTMLElement.prototype, 'clientWidth', origClientWidth);
    if (origClientHeight) Object.defineProperty(HTMLElement.prototype, 'clientHeight', origClientHeight);
  });

  it('plots each pair as a labeled dot and opens it on click', async () => {
    const onOpen = vi.fn();
    renderWithProviders(
      <RSIHeatmap
        data={{
          exchange: 'binance',
          interval: '1h',
          averageRsi: 50,
          oversoldCount: 1,
          overboughtCount: 0,
          items: [
            { rank: 1, symbol: 'BTCUSDT', base: 'BTC', rsi: 28, zone: 'oversold', marketCapCirculating: 1_000_000 },
            { rank: 40, symbol: 'SHIBUSDT', base: 'SHIB', rsi: 51, zone: 'neutral' },
            { rank: 2, base: 'NOLINK', rsi: 44, zone: 'neutral' },
          ],
        }}
        onOpen={onOpen}
      />,
    );
    expect(screen.getByTestId('rsi-heatmap')).toBeInTheDocument();
    expect(screen.getByTestId('rsi-heat-scale')).toBeInTheDocument();
    expect(screen.getByTestId('rsi-avg-line')).toBeInTheDocument();
    const fillOf = (id: string) =>
      screen.getByTestId(id).querySelector('circle[fill]:not([fill="transparent"])');
    expect(fillOf('rsi-dot-BTC')?.getAttribute('fill')).not.toBe(fillOf('rsi-dot-SHIB')?.getAttribute('fill'));
    expect(screen.getByText('BTC')).toBeInTheDocument();
    await userEvent.click(screen.getByTestId('rsi-dot-BTC'));
    expect(onOpen).toHaveBeenCalledWith('binance', 'BTCUSDT');
    await userEvent.click(screen.getByTestId('rsi-dot-NOLINK'));
    expect(onOpen).toHaveBeenCalledTimes(1);
    const map = screen.getByTestId('rsi-heatmap');
    const svg = map.querySelector('svg');
    expect(svg).toBeTruthy();
    expect(svg).not.toHaveAttribute('preserveAspectRatio', 'none');
    fireEvent.mouseEnter(screen.getByTestId('rsi-dot-BTC'), { clientX: 190, clientY: 348 });
    expect(screen.getByText(/Rank #1/)).toBeInTheDocument();
    fireEvent.mouseLeave(screen.getByTestId('rsi-dot-BTC'));
    expect(screen.queryByText(/Rank #1/)).not.toBeInTheDocument();
    fireEvent.mouseMove(svg as SVGElement, { clientX: 480, clientY: 260 });
    expect(screen.queryByText(/Rank #1/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Rank #40/)).not.toBeInTheDocument();
    fireEvent.mouseLeave(map);
  });

  it('renders without an average line and without onOpen', async () => {
    renderWithProviders(
      <RSIHeatmap
        data={{
          interval: '1h',
          items: [{ rank: 1, symbol: 'ETHUSDT', base: 'ETH', rsi: 55, zone: '' }],
        }}
        isLoading
      />,
    );
    expect(screen.getByTestId('rsi-heatmap')).toBeInTheDocument();
    expect(screen.queryByTestId('rsi-avg-line')).not.toBeInTheDocument();
    await userEvent.click(screen.getByTestId('rsi-dot-ETH'));
    fireEvent.mouseEnter(screen.getByTestId('rsi-dot-ETH'));
    expect(screen.getByText(/Rank #1/)).toBeInTheDocument();
    expect(screen.getByText('—')).toBeInTheDocument();
  });

  it('plots a row with no rank', () => {
    renderWithProviders(
      <RSIHeatmap data={{ items: [{ symbol: 'NORANKUSDT', base: 'NORANK', rsi: 48 }] }} />,
    );
    expect(screen.getByTestId('rsi-dot-NORANK')).toBeInTheDocument();
  });

  it('shows a skeleton while loading an empty map', () => {
    const { container } = renderWithProviders(<RSIHeatmap isLoading data={{ items: [] }} />);
    expect(container.querySelector('.ant-skeleton')).toBeTruthy();
    expect(screen.queryByTestId('rsi-heatmap')).not.toBeInTheDocument();
  });

  it('shows the empty desk when there are no pairs', () => {
    renderWithProviders(<RSIHeatmap />);
    expect(screen.getByText(/no markets to map/i)).toBeInTheDocument();
    expect(screen.queryByTestId('rsi-heatmap')).not.toBeInTheDocument();
  });

  it('sizes the plot to the frame after the skeleton unmounts', () => {
    Object.defineProperty(HTMLElement.prototype, 'clientWidth', {
      configurable: true,
      get: () => 1240,
    });
    const { rerender } = renderWithProviders(<RSIHeatmap isLoading data={{ items: [] }} />);
    expect(screen.queryByTestId('rsi-heatmap')).not.toBeInTheDocument();
    rerender(
      <RSIHeatmap data={{ items: [{ rank: 1, symbol: 'BTCUSDT', base: 'BTC', rsi: 40 }] }} />,
    );
    expect(screen.getByTestId('rsi-heatmap').querySelector('svg')).toHaveAttribute(
      'viewBox',
      '0 0 1240 520',
    );
  });

  it('skips ResizeObserver when the browser does not provide it', () => {
    vi.stubGlobal('ResizeObserver', undefined);
    renderWithProviders(
      <RSIHeatmap data={{ items: [{ rank: 1, symbol: 'BTCUSDT', base: 'BTC', rsi: 40 }] }} />,
    );
    expect(screen.getByTestId('rsi-heatmap')).toBeInTheDocument();
  });
});
