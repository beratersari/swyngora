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

export type ChartVertLine = {
  id: string;
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
      context.lineWidth = 1;
      context.setLineDash([5, 4]);
      context.beginPath();
      context.moveTo(x + 0.5, 0);
      context.lineTo(x + 0.5, mediaSize.height);
      context.stroke();
      if (this.label) {
        context.font = '11px system-ui, sans-serif';
        context.fillStyle = this.color;
        context.fillText(this.label, x + 6, 14 + this.labelOffsetY);
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

  constructor(opts: ChartVertLine) {
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
    const x = this._chart?.timeScale().timeToCoordinate(this._time);
    this._x = x === undefined || x === null || !Number.isFinite(x) ? null : x;
  }

  paneViews(): readonly IPrimitivePaneView[] {
    return this._paneViews;
  }

  timeAxisViews(): readonly ISeriesPrimitiveAxisView[] {
    return this._timeAxisViews;
  }

  autoscaleInfo(): null {
    return null;
  }
}

/** Bind event times to an existing OHLC bar so candles stay visible. */
export function snapVertLinesToCandleTimes<T extends { time: number }>(
  lines: readonly T[],
  candleTimes: readonly number[],
): T[] {
  const times = candleTimes.filter((t) => Number.isFinite(t) && t > 0);
  if (!lines.length || !times.length) return [];
  const sorted = times.slice().sort((a, b) => a - b);
  const timeSet = new Set(sorted);
  return lines
    .filter((line) => Number.isFinite(line.time) && line.time > 0)
    .map((line) => ({
      ...line,
      time: timeSet.has(line.time) ? line.time : nearestCandleTime(sorted, line.time),
    }));
}

function nearestCandleTime(sortedTimes: number[], target: number): number {
  let lo = 0;
  let hi = sortedTimes.length - 1;
  if (target <= sortedTimes[0]!) return sortedTimes[0]!;
  if (target >= sortedTimes[hi]!) return sortedTimes[hi]!;
  while (lo <= hi) {
    const mid = (lo + hi) >> 1;
    const v = sortedTimes[mid]!;
    if (v === target) return v;
    if (v < target) lo = mid + 1;
    else hi = mid - 1;
  }
  const a = sortedTimes[hi]!;
  const b = sortedTimes[lo]!;
  return Math.abs(a - target) <= Math.abs(b - target) ? a : b;
}

export function isoToUnixSeconds(value: string | number | Date | null | undefined): number | null {
  if (value === null || value === undefined || value === '') return null;
  const ms = value instanceof Date ? value.getTime() : Date.parse(String(value));
  if (!Number.isFinite(ms)) return null;
  return Math.floor(ms / 1000);
}
