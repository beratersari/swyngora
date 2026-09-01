import { HUNT_SCORE_FACTORS } from './constants';
import type {
  HuntCascadeStep,
  HuntCoverageLevel,
  HuntEase,
  HuntHouseEdge,
  HuntInputState,
  HuntLean,
  HuntPanel,
  HuntScenario,
  HuntVenue,
  HuntWeightDraftRow,
} from './LiquidationHunt.types';

export function parseHuntPanel(raw?: string | null): HuntPanel {
  return raw === 'path' ? 'path' : 'compare';
}

export function defaultHuntWeightDraft(): HuntWeightDraftRow[] {
  return HUNT_SCORE_FACTORS.map((f) => ({ id: f.id, enabled: true, pct: f.defaultPct }));
}

export function huntWeightTotal(draft: HuntWeightDraftRow[]): number {
  return draft.reduce((sum, row) => sum + (row.enabled ? row.pct : 0), 0);
}

export function isDefaultHuntWeightDraft(draft: HuntWeightDraftRow[]): boolean {
  const def = defaultHuntWeightDraft();
  if (draft.length !== def.length) return false;
  return def.every((d, i) => draft[i]?.id === d.id && draft[i]?.enabled === d.enabled && draft[i]?.pct === d.pct);
}

export function parseHuntWeightDraft(get: (key: string) => string | null): HuntWeightDraftRow[] | null {
  const any = HUNT_SCORE_FACTORS.some((f) => (get(f.query) ?? '').trim() !== '');
  if (!any) return null;
  return HUNT_SCORE_FACTORS.map((f) => {
    const raw = (get(f.query) ?? '').trim();
    if (raw === '') return { id: f.id, enabled: false, pct: 0 };
    const n = Number(raw);
    return { id: f.id, enabled: Number.isFinite(n) && n > 0, pct: Number.isFinite(n) ? n : 0 };
  });
}

export function huntWeightQueryParams(draft: HuntWeightDraftRow[]): Record<string, number> {
  const out: Record<string, number> = {};
  for (const factor of HUNT_SCORE_FACTORS) {
    const row = draft.find((d) => d.id === factor.id);
    out[factor.query] = row?.enabled ? row.pct : 0;
  }
  return out;
}

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

export function inputSpanText(
  have?: string,
  need?: string,
  coverPct?: number,
  age?: string,
  stale?: boolean,
): string {
  const pct = Number.isFinite(coverPct) && coverPct != null ? ` (${Math.round(coverPct)}%)` : '';
  if (stale && age && need) {
    return `${age} old / ${need}${pct}`;
  }
  if (have && need) {
    return `${have} / ${need}${pct}`;
  }
  return '';
}

export function formatEffect(raw?: number | null): string {
  const n = raw ?? 0;
  if (!Number.isFinite(n) || Math.abs(n) < 0.05) return '0';
  const abs = Math.abs(n).toFixed(1);
  return n > 0 ? `+${abs}` : `−${abs}`;
}

export function effectTone(raw?: number | null): 'up' | 'down' | 'muted' {
  const n = raw ?? 0;
  if (!Number.isFinite(n) || Math.abs(n) < 0.15) return 'muted';
  return n > 0 ? 'up' : 'down';
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

export type PathStepTone = 'start' | 'self' | 'helped' | 'stall' | 'unreachable' | 'missing';

export function pathStepTone(step?: HuntCascadeStep): PathStepTone {
  switch (step?.role) {
    case 'start':
      return 'start';
    case 'self':
      return 'self';
    case 'helped':
      return 'helped';
    case 'stall':
    case 'observed':
      return 'stall';
    case 'missing':
      return 'missing';
    case 'unreachable':
      return 'unreachable';
    default:
      break;
  }
  if (!step) return 'missing';
  if (!step.reachable) return 'unreachable';
  if (step.selfFueling) return 'self';
  if (step.easier) return 'helped';
  return 'stall';
}

export function pathLeverageLabel(step?: HuntCascadeStep): string | null {
  const lev = parseNum(step?.band?.leverage);
  if (lev == null || lev <= 0) return null;
  return `${lev}x`;
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
