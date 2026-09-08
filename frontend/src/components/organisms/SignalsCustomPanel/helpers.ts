import { CUSTOM_SIGNAL_FACTORS, type CustomSignalFactorId } from './constants';
import type { CustomSignalItem } from '@/libs/api';
import type { CustomMixPreviewFactor, CustomMixPreviewRow, CustomWeightDraftRow } from './SignalsCustomPanel.types';

export function defaultCustomWeightDraft(): CustomWeightDraftRow[] {
  return CUSTOM_SIGNAL_FACTORS.map((f) => ({
    id: f.id,
    enabled: f.defaultPct > 0,
    pct: f.defaultPct,
  }));
}

export function draftFromWeights(weights?: Record<string, number>): CustomWeightDraftRow[] {
  return CUSTOM_SIGNAL_FACTORS.map((f) => {
    const pct = Number(weights?.[f.id] ?? 0);
    return { id: f.id, enabled: pct > 0, pct: Number.isFinite(pct) ? pct : 0 };
  });
}

export function weightsFromDraft(draft: CustomWeightDraftRow[]): Record<string, number> {
  const out: Record<string, number> = {};
  for (const row of draft) {
    out[row.id] = row.enabled ? row.pct : 0;
  }
  return out;
}

export function customWeightTotal(draft: CustomWeightDraftRow[]): number {
  return draft.reduce((sum, row) => sum + (row.enabled && Number.isFinite(row.pct) ? row.pct : 0), 0);
}

export function customWeightsValid(draft: CustomWeightDraftRow[]): boolean {
  return Math.abs(customWeightTotal(draft) - 100) <= 0.05 && draft.some((r) => r.enabled && r.pct > 0);
}

export function draftNeedsHunt(draft: CustomWeightDraftRow[]): boolean {
  const row = draft.find((d) => d.id === 'hunt_up');
  return Boolean(row?.enabled && row.pct > 0);
}

export function draftNeedsTape(draft: CustomWeightDraftRow[]): boolean {
  return draft.some((d) => (d.id === 'forensics' || d.id === 'similar_setup') && d.enabled && d.pct > 0);
}

function round1(v: number): number {
  return Math.round(v * 10) / 10;
}

function clampScore(v: number): number {
  if (!Number.isFinite(v)) return 0;
  return round1(Math.max(0, Math.min(100, v)));
}

export function customSignalGrade(score: number): string {
  if (score >= 78) return 'A';
  if (score >= 62) return 'B';
  return 'C';
}

export function customEvidenceLabel(score: number): 'strong' | 'fair' | 'weak' {
  if (score >= 75) return 'strong';
  if (score >= 50) return 'fair';
  return 'weak';
}

export function previewEvidence(
  factors: CustomMixPreviewFactor[],
): {
  score: number;
  label: 'strong' | 'fair' | 'weak';
  detail: string;
  fresh: number;
  old: number;
  missing: number;
  idle: number;
} {
  let mass = 0;
  let acc = 0;
  let fresh = 0;
  let old = 0;
  let missingN = 0;
  let idle = 0;
  const missing: string[] = [];
  for (const factor of factors) {
    if (factor.appliedPct <= 0) continue;
    mass += factor.appliedPct;
    if (factor.status === 'missing') {
      acc += 0;
      missingN += 1;
      missing.push(factor.id);
      continue;
    }
    if (factor.status === 'idle') {
      acc += 0.2 * factor.appliedPct;
      idle += 1;
      continue;
    }
    if (factor.freshness === 'old') {
      old += 1;
    } else {
      fresh += 1;
    }
    const q = Number.isFinite(factor.quality) ? Math.min(1, Math.max(0, (factor.quality ?? 100) / 100)) : 1;
    acc += q * factor.appliedPct;
  }
  const score = mass > 0 ? round1(Math.max(0, Math.min(100, (acc / mass) * 100))) : 0;
  const label = customEvidenceLabel(score);
  let counts = `${fresh} fresh, ${old} old, ${missingN} missing`;
  if (idle > 0) counts += `, ${idle} not present`;
  const bits = [counts];
  if (missing.length) bits.push(`missing ${missing.join(', ')}`);
  for (const factor of factors) {
    if (factor.status !== 'used' || (factor.id !== 'forensics' && factor.id !== 'similar_setup')) continue;
    const note = tapeStatsPrefix(factor.detail);
    if (note) {
      bits.push(note);
      break;
    }
  }
  return { score, label, detail: `Evidence is ${label} (${Math.round(score)}) — ${bits.join('; ')}`, fresh, old, missing: missingN, idle };
}

function tapeStatsPrefix(detail?: string): string {
  const text = (detail ?? '').trim();
  if (!text.startsWith('Used ')) return '';
  const cut = text.indexOf('. ');
  return cut > 0 ? text.slice(0, cut) : text;
}

function pctOf(draft: CustomWeightDraftRow[], id: CustomSignalFactorId): number {
  const row = draft.find((d) => d.id === id);
  if (!row?.enabled) return 0;
  return Number.isFinite(row.pct) ? row.pct : 0;
}

function assemble(
  item: CustomSignalItem | undefined,
  pct: (id: CustomSignalFactorId) => number,
): { score: number; effects: Map<string, number>; statuses: Map<string, CustomMixPreviewFactor['status']> } {
  const effects = new Map<string, number>();
  const statuses = new Map<string, CustomMixPreviewFactor['status']>();
  const live: { id: CustomSignalFactorId; pct: number; score: number }[] = [];
  for (const factor of CUSTOM_SIGNAL_FACTORS) {
    const src = item?.custom?.factors?.find((f) => f.id === factor.id);
    const share = pct(factor.id);
    if (share <= 0) {
      statuses.set(factor.id, 'disabled');
      effects.set(factor.id, 0);
      continue;
    }
    if (src?.status === 'missing') {
      statuses.set(factor.id, 'missing');
      effects.set(factor.id, 0);
      continue;
    }
    if (src?.status === 'idle' || !Number.isFinite(src?.score) || (src?.score ?? 0) <= 0) {
      statuses.set(factor.id, 'idle');
      effects.set(factor.id, 0);
      continue;
    }
    statuses.set(factor.id, 'used');
    live.push({ id: factor.id, pct: share, score: src?.score ?? 0 });
  }
  let mass = 0;
  let acc = 0;
  for (const row of live) {
    mass += row.pct;
    acc += row.score * row.pct;
  }
  let score = 0;
  if (mass > 0) {
    score = acc / mass;
    const breadth = Math.min(1, live.length / 5);
    score *= 0.9 + 0.1 * breadth;
    for (const row of live) {
      effects.set(row.id, round1((row.score - 50) * (row.pct / mass)));
    }
  }
  return { score: clampScore(score), effects, statuses };
}

export function largestCustomFactorChange(factors: CustomMixPreviewFactor[]): CustomMixPreviewFactor | null {
  let best: CustomMixPreviewFactor | null = null;
  for (const factor of factors) {
    if (Math.abs(factor.deltaEffect) < 0.05) continue;
    if (!best || Math.abs(factor.deltaEffect) > Math.abs(best.deltaEffect)) {
      best = factor;
    }
  }
  return best;
}

export function previewCustomMix(items: CustomSignalItem[], draft: CustomWeightDraftRow[]): CustomMixPreviewRow[] {
  const def = defaultCustomWeightDraft();
  const defaultPct = (id: CustomSignalFactorId) => pctOf(def, id);
  const appliedPct = (id: CustomSignalFactorId) => pctOf(draft, id);
  return items.map((item) => {
    const assembledDef = assemble(item, defaultPct);
    const assembledApp = assemble(item, appliedPct);
    const factors: CustomMixPreviewFactor[] = CUSTOM_SIGNAL_FACTORS.map((factor) => {
      const src = item.custom?.factors?.find((f) => f.id === factor.id);
      return {
        id: factor.id,
        score: src?.score ?? 0,
        status: assembledApp.statuses.get(factor.id) ?? 'missing',
        defaultPct: defaultPct(factor.id),
        appliedPct: appliedPct(factor.id),
        defaultEffect: assembledDef.effects.get(factor.id) ?? 0,
        appliedEffect: assembledApp.effects.get(factor.id) ?? 0,
        deltaEffect: round1((assembledApp.effects.get(factor.id) ?? 0) - (assembledDef.effects.get(factor.id) ?? 0)),
        detail: src?.detail,
        quality: src?.quality,
        freshness: src?.freshness,
      };
    });
    const hunt = factors.find((f) => f.id === 'hunt_up');
    const tapeMissing = factors.some(
      (f) => (f.id === 'forensics' || f.id === 'similar_setup') && f.appliedPct > 0 && f.status === 'missing',
    );
    const evidence = previewEvidence(factors);
    return {
      exchange: item.exchange ?? '',
      symbol: item.symbol ?? '',
      interval: item.interval,
      price: item.price,
      defaultScore: Number.isFinite(item.engine?.swingScore)
        ? round1(item.engine?.swingScore ?? assembledDef.score)
        : assembledDef.score,
      appliedScore: assembledApp.score,
      delta: round1(
        assembledApp.score -
          (Number.isFinite(item.engine?.swingScore)
            ? (item.engine?.swingScore ?? assembledDef.score)
            : assembledDef.score),
      ),
      grade: customSignalGrade(assembledApp.score),
      factors,
      largestChange: largestCustomFactorChange(factors),
      huntMissing: Boolean(hunt && hunt.appliedPct > 0 && hunt.status === 'missing'),
      tapeMissing,
      tapeSetups: item.tapeSetups,
      tapeUsed: item.tapeUsed,
      tapeThin: item.tapeThin,
      tapeFreshNote: item.tapeFresh?.note,
      tapeFreshStale: Boolean(
        item.tapeFresh?.oi?.stale ||
          item.tapeFresh?.takers?.stale ||
          item.tapeFresh?.book?.stale ||
          item.tapeFresh?.options?.stale,
      ),
      evidenceScore: evidence.score,
      evidenceLabel: evidence.label,
      evidenceDetail: evidence.detail,
      evidenceFresh: evidence.fresh,
      evidenceOld: evidence.old,
      evidenceMissing: evidence.missing,
      evidenceIdle: evidence.idle,
    };
  }).sort((a, b) => b.appliedScore - a.appliedScore || a.symbol.localeCompare(b.symbol));
}

export function scanWeightParams(draft: CustomWeightDraftRow[]): Record<string, number> {
  const out: Record<string, number> = {};
  for (const factor of CUSTOM_SIGNAL_FACTORS) {
    const row = draft.find((d) => d.id === factor.id);
    out[factor.query] = row?.enabled ? row.pct : 0;
  }
  return out;
}
