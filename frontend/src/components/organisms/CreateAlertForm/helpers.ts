import { SITUATION_MATCH_CLAUSES, type SituationMatch } from './constants';

/** Drop empty clauses. Undefined means nothing is selected. */
export function compactSituationMatch(match: SituationMatch): SituationMatch | undefined {
  const out: SituationMatch = {};
  for (const clause of SITUATION_MATCH_CLAUSES) {
    const value = (match[clause.id] ?? '').trim();
    if (value && (clause.kinds as readonly string[]).includes(value)) out[clause.id] = value;
  }
  return Object.keys(out).length > 0 ? out : undefined;
}

/** Known clause kinds from the alerts page query. Unknown tokens are ignored. */
export function situationMatchFromSearch(params: { get(name: string): string | null }): SituationMatch {
  const out: SituationMatch = {};
  for (const clause of SITUATION_MATCH_CLAUSES) {
    const value = (params.get(clause.id) ?? '').trim();
    if ((clause.kinds as readonly string[]).includes(value)) out[clause.id] = value;
  }
  return out;
}
