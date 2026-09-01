export const LIQ_HUNT_POLL_MS = 20_000;

export const HUNT_SCORE_FACTORS = [
  { id: 'proximity', query: 'weightProximity', defaultPct: 20 },
  { id: 'book', query: 'weightBook', defaultPct: 16 },
  { id: 'efficiency', query: 'weightEfficiency', defaultPct: 12 },
  { id: 'trend', query: 'weightTrend', defaultPct: 20 },
  { id: 'crowding', query: 'weightCrowding', defaultPct: 18 },
  { id: 'flow', query: 'weightFlow', defaultPct: 14 },
] as const;

export type HuntScoreFactorId = (typeof HUNT_SCORE_FACTORS)[number]['id'];
