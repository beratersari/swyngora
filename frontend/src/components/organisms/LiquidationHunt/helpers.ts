import { HUNT_SCORE_FACTORS } from './constants';
import type {
  HuntCascadeStep,
  HuntCoverageLevel,
  HuntDirectionScore,
  HuntEase,
  HuntFactor,
  HuntHouseEdge,
  HuntInputState,
  HuntLean,
  HuntMixPreview,
  HuntMixPreviewFactor,
  HuntPanel,
  HuntScenario,
  HuntVenue,
  HuntWeightDraftRow,
} from './LiquidationHunt.types';

/** Matches backend huntScoreKeep: 0.30 + 0.70 × coverage. */
export function huntScoreKeep(coverage?: number): number {
  const c = Math.max(0, Math.min(1, scoreValue(coverage ?? 100) / 100));
  return 0.3 + 0.7 * c;
}

function round1(v: number): number {
  return Math.round(v * 10) / 10;
}

function clampScore(v: number): number {
  if (!Number.isFinite(v)) return 0;
  return round1(Math.max(0, Math.min(100, v)));
}

export function huntLeanFromScores(up: number, down: number): { lean: HuntLean; margin: number } {
  const margin = round1(Math.abs(up - down));
  if (up - down >= 8) return { lean: 'up', margin };
  if (down - up >= 8) return { lean: 'down', margin };
  return { lean: 'even', margin };
}

function factorById(score: HuntDirectionScore | undefined, id: string): HuntFactor | undefined {
  return score?.factors?.find((f) => f.id === id);
}

function factorHasData(factor?: HuntFactor): boolean {
  if (!factor) return false;
  if (factor.status === 'missing') return false;
  return factor.status === 'used' || factor.status === 'disabled' || Number.isFinite(factor.score);
}

function mixPct(draft: HuntWeightDraftRow[], id: string): number {
  const row = draft.find((d) => d.id === id);
  if (!row?.enabled) return 0;
  return Number.isFinite(row.pct) ? row.pct : 0;
}

function assembleDirection(
  score: HuntDirectionScore | undefined,
  pctOf: (id: string) => number,
  coverage?: number,
): { score: number; effects: Map<string, number>; statuses: Map<string, HuntMixPreviewFactor['status']> } {
  const keep = huntScoreKeep(coverage);
  let raw = 50;
  const effects = new Map<string, number>();
  const statuses = new Map<string, HuntMixPreviewFactor['status']>();
  for (const factor of HUNT_SCORE_FACTORS) {
    const src = factorById(score, factor.id);
    const pct = pctOf(factor.id);
    if (pct <= 0) {
      statuses.set(factor.id, 'disabled');
      effects.set(factor.id, 0);
      continue;
    }
    if (!factorHasData(src)) {
      statuses.set(factor.id, 'missing');
      effects.set(factor.id, 0);
      continue;
    }
    const rawScore = src?.score ?? 50;
    raw += (rawScore - 50) * (pct / 100);
    statuses.set(factor.id, 'used');
    effects.set(factor.id, round1((rawScore - 50) * (pct / 100) * keep));
  }
  const clamped = clampScore(raw);
  return { score: clampScore(50 + (clamped - 50) * keep), effects, statuses };
}

function previewOneVenue(venue: HuntVenue, draft: HuntWeightDraftRow[]): HuntMixPreview {
  const coverage = venue.coverage?.score ?? 100;
  const defaultPct = (id: string) => HUNT_SCORE_FACTORS.find((f) => f.id === id)?.defaultPct ?? 0;
  const appliedPct = (id: string) => mixPct(draft, id);
  const defUp = assembleDirection(venue.upScore, defaultPct, coverage);
  const defDown = assembleDirection(venue.downScore, defaultPct, coverage);
  const appUp = assembleDirection(venue.upScore, appliedPct, coverage);
  const appDown = assembleDirection(venue.downScore, appliedPct, coverage);
  const defLean = huntLeanFromScores(defUp.score, defDown.score);
  const appLean = huntLeanFromScores(appUp.score, appDown.score);
  const factors: HuntMixPreviewFactor[] = HUNT_SCORE_FACTORS.map((factor) => {
    const src = factorById(venue.upScore, factor.id);
    return {
      id: factor.id,
      defaultPct: defaultPct(factor.id),
      appliedPct: appliedPct(factor.id),
      score: src?.score ?? 0,
      status: appUp.statuses.get(factor.id) ?? 'missing',
      defaultEffect: defUp.effects.get(factor.id) ?? 0,
      appliedEffect: appUp.effects.get(factor.id) ?? 0,
      deltaEffect: round1((appUp.effects.get(factor.id) ?? 0) - (defUp.effects.get(factor.id) ?? 0)),
    };
  });
  return {
    exchange: venue.exchange,
    coverage,
    defaultUp: defUp.score,
    defaultDown: defDown.score,
    appliedUp: appUp.score,
    appliedDown: appDown.score,
    defaultLean: defLean.lean,
    appliedLean: appLean.lean,
    upDelta: round1(appUp.score - defUp.score),
    downDelta: round1(appDown.score - defDown.score),
    factors,
  };
}

export function previewHuntMix(venues: HuntVenue[], draft: HuntWeightDraftRow[]): HuntMixPreview | null {
  const usable = venues.filter((v) => v.coverage?.usable !== false && !v.error);
  const rows = (usable.length > 0 ? usable : venues).map((v) => previewOneVenue(v, draft));
  if (rows.length === 0) return null;
  if (rows.length === 1) return rows[0];
  let oiSum = 0;
  let defUp = 0;
  let defDown = 0;
  let appUp = 0;
  let appDown = 0;
  const factorAcc = new Map<string, HuntMixPreviewFactor & { w: number }>();
  for (const row of rows) {
    const venue = venues.find((v) => v.exchange === row.exchange);
    const w = parseNum(venue?.openInterestValue) ?? 1;
    oiSum += w;
    defUp += row.defaultUp * w;
    defDown += row.defaultDown * w;
    appUp += row.appliedUp * w;
    appDown += row.appliedDown * w;
    for (const f of row.factors) {
      const acc = factorAcc.get(f.id);
      if (!acc) {
        factorAcc.set(f.id, { ...f, w });
        continue;
      }
      acc.score = (acc.score * acc.w + f.score * w) / (acc.w + w);
      acc.defaultEffect = (acc.defaultEffect * acc.w + f.defaultEffect * w) / (acc.w + w);
      acc.appliedEffect = (acc.appliedEffect * acc.w + f.appliedEffect * w) / (acc.w + w);
      acc.deltaEffect = (acc.deltaEffect * acc.w + f.deltaEffect * w) / (acc.w + w);
      if (f.status === 'used') acc.status = 'used';
      acc.w += w;
    }
  }
  if (oiSum <= 0) return rows[0];
  const defaultUp = clampScore(defUp / oiSum);
  const defaultDown = clampScore(defDown / oiSum);
  const appliedUp = clampScore(appUp / oiSum);
  const appliedDown = clampScore(appDown / oiSum);
  return {
    coverage: rows[0]?.coverage ?? 100,
    defaultUp,
    defaultDown,
    appliedUp,
    appliedDown,
    defaultLean: huntLeanFromScores(defaultUp, defaultDown).lean,
    appliedLean: huntLeanFromScores(appliedUp, appliedDown).lean,
    upDelta: round1(appliedUp - defaultUp),
    downDelta: round1(appliedDown - defaultDown),
    factors: HUNT_SCORE_FACTORS.map((factor) => {
      const acc = factorAcc.get(factor.id);
      if (!acc) {
        return {
          id: factor.id,
          defaultPct: factor.defaultPct,
          appliedPct: mixPct(draft, factor.id),
          score: 0,
          status: 'missing' as const,
          defaultEffect: 0,
          appliedEffect: 0,
          deltaEffect: 0,
        };
      }
      return {
        id: acc.id,
        defaultPct: acc.defaultPct,
        appliedPct: acc.appliedPct,
        score: round1(acc.score),
        status: acc.status,
        defaultEffect: round1(acc.defaultEffect),
        appliedEffect: round1(acc.appliedEffect),
        deltaEffect: round1(acc.deltaEffect),
      };
    }),
  };
}

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
