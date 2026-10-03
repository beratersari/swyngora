import type {
  IChartApi,
  IPrimitivePaneRenderer,
  IPrimitivePaneView,
  ISeriesApi,
  ISeriesPrimitive,
  SeriesAttachedParameter,
  SeriesType,
  Time,
} from 'lightweight-charts';
import { semanticColors } from '@/styles/tokens';
import type {
  CandleChartFootprintLevel,
  CandleChartHeatColumn,
  CandleChartLiqBar,
  CandleChartLiqCell,
} from './CandleChartHost.types';

export type DeskOverlayModel = {
  bookHeat: CandleChartHeatColumn[];
  footprint: CandleChartFootprintLevel[];
  liqProfile: CandleChartLiqBar[];
  liqHeat: CandleChartLiqCell[];
};

type MediaTarget = {
  useMediaCoordinateSpace: (
    fn: (scope: {
      context: CanvasRenderingContext2D;
      mediaSize: { width: number; height: number };
    }) => void,
  ) => void;
};

const EMPTY: DeskOverlayModel = { bookHeat: [], footprint: [], liqProfile: [], liqHeat: [] };

const LIQ_HEAT_ALPHA = 0.72;

type TimeScaleX = {
  timeToCoordinate: (time: Time) => number | null;
  getVisibleRange: () => { from: Time; to: Time } | null;
};

type SpanX = { from: number; xFrom: number; pxPerSec: number };

/** Pixels per second across the candles currently on screen. */
function visibleSpan(scale: TimeScaleX): SpanX | null {
  const range = scale.getVisibleRange();
  if (!range || typeof range.from !== 'number' || typeof range.to !== 'number' || range.to === range.from) {
    return null;
  }
  const xFrom = scale.timeToCoordinate(range.from);
  const xTo = scale.timeToCoordinate(range.to);
  if (xFrom == null || xTo == null) return null;
  return { from: range.from, xFrom, pxPerSec: (xTo - xFrom) / (range.to - range.from) };
}

/**
 * Map a unix second onto the candle scale.
 * Exact candle opens use the scale. Other times, including heatmap columns,
 * use the visible span so a 30-minute column still sits on 1-hour candles.
 */
function timeToX(scale: TimeScaleX, sec: number, span: SpanX | null): number | null {
  const exact = scale.timeToCoordinate(sec as Time);
  if (exact != null) return exact;
  if (!span) return null;
  return span.xFrom + (sec - span.from) * span.pxPerSec;
}

function heatAlpha(notional: number, max: number): number {
  if (max <= 0 || notional <= 0) return 0;
  return 0.12 + 0.38 * Math.min(1, notional / max);
}

class HeatRenderer implements IPrimitivePaneRenderer {
  constructor(
    private readonly chart: () => IChartApi | null,
    private readonly series: () => ISeriesApi<SeriesType> | null,
    private readonly model: () => DeskOverlayModel,
  ) {}

  draw(target: MediaTarget): void {
    const chart = this.chart();
    const series = this.series();
    const cols = this.model().bookHeat;
    const cells = this.model().liqHeat;
    if (!chart || !series || (cols.length === 0 && cells.length === 0)) return;
    target.useMediaCoordinateSpace(({ context, mediaSize }) => {
      if (cells.length > 0) {
        const span = visibleSpan(chart.timeScale());
        const scale = chart.timeScale();
        context.save();
        context.globalAlpha = LIQ_HEAT_ALPHA;
        for (const cell of cells) {
          const x1 = timeToX(scale, cell.time, span);
          const x2 = timeToX(scale, cell.time + cell.stepSec, span);
          if (x1 == null || x2 == null) continue;
          const left = Math.min(x1, x2);
          const width = Math.max(1, Math.abs(x2 - x1));
          if (left > mediaSize.width || left + width < 0) continue;
          const yTop = series.priceToCoordinate(cell.priceHi);
          const yBot = series.priceToCoordinate(cell.priceLo);
          if (yTop == null || yBot == null) continue;
          const top = Math.min(yTop, yBot);
          const height = Math.max(1, Math.abs(yBot - yTop));
          if (top > mediaSize.height || top + height < 0) continue;
          context.fillStyle = cell.color;
          context.fillRect(left, top, width, height);
        }
        context.restore();
      }
      if (cols.length === 0) return;
      let max = 0;
      for (const col of cols) {
        for (const lv of col.bids) max = Math.max(max, lv.notional);
        for (const lv of col.asks) max = Math.max(max, lv.notional);
      }
      if (max <= 0) return;
      context.save();
      const scale = chart.timeScale();
      for (let i = 0; i < cols.length; i += 1) {
        const col = cols[i]!;
        const x1 = scale.timeToCoordinate(col.time as Time);
        const next = cols[i + 1];
        const x2 = next ? scale.timeToCoordinate(next.time as Time) : null;
        if (x1 == null) continue;
        const width = x2 != null && x2 > x1 ? Math.max(1, x2 - x1) : 6;
        const paint = (levels: { price: number; notional: number }[], color: string) => {
          const sorted = [...levels].sort((a, b) => b.price - a.price);
          for (let n = 0; n < sorted.length; n += 1) {
            const lv = sorted[n]!;
            const y = series.priceToCoordinate(lv.price);
            if (y == null) continue;
            const below = sorted[n + 1] ? series.priceToCoordinate(sorted[n + 1]!.price) : null;
            const h = below != null && below > y ? Math.max(1, below - y) : 4;
            context.globalAlpha = heatAlpha(lv.notional, max);
            context.fillStyle = color;
            context.fillRect(x1, y, width, h);
          }
        };
        paint(col.bids, semanticColors.chart.up);
        paint(col.asks, semanticColors.chart.down);
      }
      context.restore();
    });
  }
}

class ProfileRenderer implements IPrimitivePaneRenderer {
  constructor(
    private readonly series: () => ISeriesApi<SeriesType> | null,
    private readonly model: () => DeskOverlayModel,
  ) {}

  draw(target: MediaTarget): void {
    const series = this.series();
    const { footprint, liqProfile } = this.model();
    if (!series || (footprint.length === 0 && liqProfile.length === 0)) return;
    target.useMediaCoordinateSpace(({ context, mediaSize }) => {
      context.save();
      const right = mediaSize.width - 4;
      if (liqProfile.length > 0) {
        let max = 0;
        for (const bar of liqProfile) max = Math.max(max, bar.notional);
        if (max > 0) {
          for (const bar of liqProfile) {
            const y = series.priceToCoordinate(bar.price);
            if (y == null) continue;
            const w = 4 + 28 * (bar.notional / max);
            context.globalAlpha = 0.85;
            context.fillStyle = bar.side === 'long' ? semanticColors.chart.down : semanticColors.chart.up;
            context.fillRect(right - 36 - w, y - 2, w, 4);
          }
        }
      }
      if (footprint.length > 0) {
        let max = 0;
        for (const lv of footprint) max = Math.max(max, lv.buy + lv.sell);
        if (max > 0) {
          for (const lv of footprint) {
            const y = series.priceToCoordinate(lv.price);
            if (y == null) continue;
            const total = lv.buy + lv.sell;
            const w = 4 + 48 * (total / max);
            const buyW = total > 0 ? w * (lv.buy / total) : 0;
            context.globalAlpha = 0.9;
            context.fillStyle = semanticColors.chart.down;
            context.fillRect(right - w, y - 3, w - buyW, 6);
            context.fillStyle = semanticColors.chart.up;
            context.fillRect(right - buyW, y - 3, buyW, 6);
          }
        }
      }
      context.restore();
    });
  }
}

class HeatView implements IPrimitivePaneView {
  private readonly _renderer: HeatRenderer;
  constructor(renderer: HeatRenderer) {
    this._renderer = renderer;
  }
  zOrder(): 'bottom' {
    return 'bottom';
  }
  renderer(): IPrimitivePaneRenderer {
    return this._renderer;
  }
}

class ProfileView implements IPrimitivePaneView {
  private readonly _renderer: ProfileRenderer;
  constructor(renderer: ProfileRenderer) {
    this._renderer = renderer;
  }
  zOrder(): 'top' {
    return 'top';
  }
  renderer(): IPrimitivePaneRenderer {
    return this._renderer;
  }
}

/** Book heatmap behind candles, plus footprint and liquidation profile on the right. */
export class DeskOverlayPrimitive implements ISeriesPrimitive<Time> {
  private _chart: IChartApi | null = null;
  private _series: ISeriesApi<SeriesType> | null = null;
  private _requestUpdate: (() => void) | null = null;
  private _model: DeskOverlayModel = EMPTY;
  private readonly _views: readonly IPrimitivePaneView[];

  constructor() {
    const chart = () => this._chart;
    const series = () => this._series;
    const model = () => this._model;
    this._views = [
      new HeatView(new HeatRenderer(chart, series, model)),
      new ProfileView(new ProfileRenderer(series, model)),
    ];
  }

  attached(param: SeriesAttachedParameter<Time, SeriesType>): void {
    this._chart = param.chart;
    this._series = param.series;
    const extra = param as SeriesAttachedParameter<Time, SeriesType> & { requestUpdate?: () => void };
    this._requestUpdate = extra.requestUpdate ?? null;
  }

  detached(): void {
    this._chart = null;
    this._series = null;
    this._requestUpdate = null;
  }

  updateAllViews(): void {}

  paneViews(): readonly IPrimitivePaneView[] {
    return this._views;
  }

  setModel(model: DeskOverlayModel): void {
    this._model = model;
    this._requestUpdate?.();
  }
}

export { heatAlpha };
