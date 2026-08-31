import type {
  HuntCoverageLevel,
  HuntEase,
  HuntHouseEdge,
  HuntInputState,
  HuntLean,
  HuntScenario,
  HuntVenue,
} from './LiquidationHunt.types';

export function leanTone(lean?: HuntLean): 'up' | 'down' | 'even' {
  if (lean === 'up') return 'up';
  if (lean === 'down') return 'down';
  return 'even';
}

export function easeTone(level?: HuntEase): 'easier' | 'likely' | 'mixed' | 'hard' {
  if (level === 'easier' || level === 'likely' || level === 'mixed' || level === 'hard') {
    return level;
  }
  return 'mixed';
}

export function houseTone(edge?: HuntHouseEdge): 'profit' | 'loss' | 'unreachable' {
  if (edge === 'profit' || edge === 'loss' || edge === 'unreachable') return edge;
  return 'unreachable';
}

export function coverageTone(
  level?: HuntCoverageLevel,
): 'complete' | 'usable' | 'thin' | 'insufficient' {
  if (level === 'complete' || level === 'usable' || level === 'thin' || level === 'insufficient') {
    return level;
  }
  return 'thin';
}

export function inputTone(status?: HuntInputState): 'ok' | 'weak' | 'missing' | 'error' {
  if (status === 'ok' || status === 'weak' || status === 'missing' || status === 'error') {
    return status;
  }
  return 'missing';
}

export function venueLabel(exchange?: string): string {
  const v = (exchange ?? '').toLowerCase();
  if (v === 'binance') return 'Binance';
  if (v === 'bybit') return 'Bybit';
  return exchange || 'Venue';
}

export function parseNum(raw?: string | number | null): number | null {
  if (raw == null || raw === '') return null;
  const n = typeof raw === 'number' ? raw : Number(raw);
  return Number.isFinite(n) ? n : null;
}

export function scoreValue(raw?: number | null): number {
  const n = raw ?? 0;
  if (!Number.isFinite(n)) return 0;
  return Math.max(0, Math.min(100, n));
}

export type HuntMetricRow = {
  id: string;
  up: string;
  down: string;
  upTone?: 'up' | 'down' | 'muted' | 'profit' | 'loss';
  downTone?: 'up' | 'down' | 'muted' | 'profit' | 'loss';
};

export function compareRows(
  venue: HuntVenue,
  formatMoney: (v: string | number | null | undefined) => string,
): HuntMetricRow[] {
  const up = venue.upHunt;
  const down = venue.downHunt;
  return [
    {
      id: 'target',
      up: formatTarget(up),
      down: formatTarget(down),
    },
    {
      id: 'spot',
      up: formatMoney(up?.spot?.notional),
      down: formatMoney(down?.spot?.notional),
      upTone: cheaperTone(up?.spot?.notional, down?.spot?.notional, 'up'),
      downTone: cheaperTone(down?.spot?.notional, up?.spot?.notional, 'down'),
    },
    {
      id: 'liq',
      up: formatMoney(up?.estLiquidated),
      down: formatMoney(down?.estLiquidated),
    },
    {
      id: 'efficiency',
      up: formatMult(up?.efficiency),
      down: formatMult(down?.efficiency),
    },
    {
      id: 'desk',
      up: formatSigned(up?.netWithCascade, formatMoney),
      down: formatSigned(down?.netWithCascade, formatMoney),
      upTone: houseTone(up?.houseEdge) === 'profit' ? 'profit' : houseTone(up?.houseEdge) === 'loss' ? 'loss' : 'muted',
      downTone:
        houseTone(down?.houseEdge) === 'profit' ? 'profit' : houseTone(down?.houseEdge) === 'loss' ? 'loss' : 'muted',
    },
    {
      id: 'book',
      up: reachLabel(up),
      down: reachLabel(down),
      upTone: up?.spot?.reachable ? 'up' : 'muted',
      downTone: down?.spot?.reachable ? 'down' : 'muted',
    },
  ];
}

export function formatTarget(sc?: HuntScenario): string {
  const px = sc?.target?.price;
  const move = sc?.target?.movePct;
  if (!px) return '—';
  if (!move) return px;
  const n = parseNum(move);
  const signed = n == null ? move : `${n > 0 ? '+' : ''}${n.toFixed(2)}%`;
  return `${px} (${signed})`;
}

function formatMult(raw?: string): string {
  const n = parseNum(raw);
  if (n == null) return '—';
  return `${n.toFixed(2)}×`;
}

function formatSigned(
  raw: string | undefined,
  formatMoney: (v: string | number | null | undefined) => string,
): string {
  const n = parseNum(raw);
  if (n == null) return '—';
  const abs = formatMoney(Math.abs(n));
  if (n > 0) return `+${abs}`;
  if (n < 0) return `−${abs}`;
  return abs;
}

function reachLabel(sc?: HuntScenario): string {
  if (!sc?.spot) return '—';
  if (sc.houseEdge === 'unreachable' && !sc.spot.reachable) return 'Unreachable';
  if (sc.spot.exhausted) return 'Visible book only';
  if (sc.spot.reachable) return 'Reachable';
  return 'Unreachable';
}

function cheaperTone(
  self?: string,
  other?: string,
  side: 'up' | 'down' = 'up',
): 'up' | 'down' | 'muted' {
  const a = parseNum(self);
  const b = parseNum(other);
  if (a == null || b == null || a <= 0 || b <= 0) return 'muted';
  if (a < b * 0.92) return side;
  return 'muted';
}
