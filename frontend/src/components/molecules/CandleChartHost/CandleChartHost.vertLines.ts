import type {
  IChartApi,
  IPrimitivePaneRenderer,
  IPrimitivePaneView,
  ISeriesPrimitive,
  ISeriesPrimitiveAxisView,
  SeriesAttachedParameter,
  SeriesType,
  Time,
} from 'lightweight-charts';

export type CandleChartVertLine = {
  id: string;
  /** Unix seconds (UTC). */
  time: number;
  color: string;
  label: string;
  /** Extra pixels from the top for stacked labels on the same bar. */
  labelOffsetY?: number;
};

type MediaTarget = {
  useMediaCoordinateSpace: (
    fn: (scope: {
      context: CanvasRenderingContext2D;
      mediaSize: { width: number; height: number };
    }) => void,
  ) => void;
};

class VertLineRenderer implements IPrimitivePaneRenderer {
  constructor(
    private readonly getX: () => number | null,
    private readonly color: string,
    private readonly label: string,
    private readonly labelOffsetY: number,
  ) {}

  draw(target: MediaTarget): void {
    target.useMediaCoordinateSpace(({ context, mediaSize }) => {
      const x = this.getX();
      if (x === null || !Number.isFinite(x)) return;
      context.save();
      context.strokeStyle = this.color;
      context.globalAlpha = 0.9;
      context.lineWidth = 1;
      context.setLineDash([5, 4]);
      context.beginPath();
      context.moveTo(x + 0.5, 0);
      context.lineTo(x + 0.5, mediaSize.height);
      context.stroke();
      if (this.label) {
        context.setLineDash([]);
        context.font = '11px Inter, "Segoe UI", system-ui, sans-serif';
        const padX = 4;
        const padY = 3;
        const w = context.measureText(this.label).width;
        const boxW = w + padX * 2;
        const boxH = 16;
        const boxX = Math.min(Math.max(x + 6, 2), mediaSize.width - boxW - 2);
        const boxY = 6 + this.labelOffsetY;
        context.fillStyle = this.color;
        context.fillRect(boxX, boxY, boxW, boxH);
        context.fillStyle = '#FFFFFF';
        context.fillText(this.label, boxX + padX, boxY + boxH - padY - 1);
      }
      context.restore();
    });
  }
}

class VertLinePaneView implements IPrimitivePaneView {
  private readonly _renderer: VertLineRenderer;

  constructor(getX: () => number | null, color: string, label: string, labelOffsetY: number) {
    this._renderer = new VertLineRenderer(getX, color, label, labelOffsetY);
  }

  zOrder(): 'top' {
    return 'top';
  }

  renderer(): IPrimitivePaneRenderer {
    return this._renderer;
  }
}

class VertLineTimeAxisView implements ISeriesPrimitiveAxisView {
  constructor(
    private readonly getX: () => number | null,
    private readonly color: string,
    private readonly label: string,
  ) {}

  coordinate(): number {
    return this.getX() ?? -10_000;
  }

  text(): string {
    return this.label;
  }

  textColor(): string {
    return '#FFFFFF';
  }

  backColor(): string {
    return this.color;
  }

  visible(): boolean {
    return this.getX() !== null;
  }
}

export class VertLinePrimitive implements ISeriesPrimitive<Time> {
  private _chart: IChartApi | null = null;
  private _x: number | null = null;
  private readonly _time: Time;
  private readonly _paneViews: VertLinePaneView[];
  private readonly _timeAxisViews: VertLineTimeAxisView[];

  constructor(private readonly opts: CandleChartVertLine) {
    this._time = opts.time as Time;
    const getX = () => this._x;
    this._paneViews = [
      new VertLinePaneView(getX, opts.color, opts.label, opts.labelOffsetY ?? 0),
    ];
    this._timeAxisViews = [new VertLineTimeAxisView(getX, opts.color, opts.label)];
  }

  attached(param: SeriesAttachedParameter<Time, SeriesType>): void {
    this._chart = param.chart;
  }

  detached(): void {
    this._chart = null;
    this._x = null;
  }

  updateAllViews(): void {
    const chart = this._chart;
    if (!chart) {
      this._x = null;
      return;
    }
    const x = chart.timeScale().timeToCoordinate(this._time);
    this._x = x === undefined || x === null || !Number.isFinite(x) ? null : x;
  }

  paneViews(): readonly IPrimitivePaneView[] {
    return this._paneViews;
  }

  timeAxisViews(): readonly ISeriesPrimitiveAxisView[] {
    return this._timeAxisViews;
  }

  /** Never participate in price autoscale — a stray NaN here hides candles. */
  autoscaleInfo(): null {
    return null;
  }
}

export type SnapVertLineOpts = {
  /** Bar length in seconds (1m=60, 1h=3600, 1d=86400, …). */
  barDurationSec?: number;
};

/**
 * Bind an event to the OHLC bar that contains it (latest open ≤ event).
 * Nearest-open would move a late-bar halt (e.g. 20:00 on a 1d) onto the next candle.
 * Events before the first bar, or after the last bar closes, are dropped.
 */
export function snapVertLinesToCandleTimes<T extends { time: number }>(
  lines: readonly T[],
  candleTimes: readonly number[],
  opts?: SnapVertLineOpts,
): T[] {
  const times = candleTimes.filter((t) => Number.isFinite(t) && t > 0);
  if (!lines.length || !times.length) return [];
  const sorted = times.slice().sort((a, b) => a - b);
  const barSec = opts?.barDurationSec && opts.barDurationSec > 0 ? opts.barDurationSec : 0;
  const out: T[] = [];
  for (const line of lines) {
    if (!Number.isFinite(line.time) || line.time <= 0) continue;
    const open = containingCandleOpen(sorted, line.time, barSec);
    if (open == null) continue;
    out.push({ ...line, time: open });
  }
  return out;
}

/** Latest candle open that contains `target`, or null if that bar is not loaded. */
export function containingCandleOpen(
  sortedOpens: readonly number[],
  target: number,
  barDurationSec = 0,
): number | null {
  if (!sortedOpens.length) return null;
  const first = sortedOpens[0]!;
  if (target < first) return null;
  let lo = 0;
  let hi = sortedOpens.length - 1;
  while (lo <= hi) {
    const mid = (lo + hi) >> 1;
    const v = sortedOpens[mid]!;
    if (v === target) return v;
    if (v < target) lo = mid + 1;
    else hi = mid - 1;
  }
  const open = sortedOpens[hi];
  if (open == null) return null;
  if (barDurationSec > 0 && target >= open + barDurationSec) return null;
  return open;
}

export function isoToUnixSeconds(value: string | number | Date | null | undefined): number | null {
  if (value === null || value === undefined || value === '') return null;
  const ms = value instanceof Date ? value.getTime() : Date.parse(String(value));
  if (!Number.isFinite(ms)) return null;
  return Math.floor(ms / 1000);
}
