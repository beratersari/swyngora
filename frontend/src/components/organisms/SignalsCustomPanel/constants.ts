export const CUSTOM_SIGNAL_FACTORS = [
  { id: 'ema_crossover', query: 'weightEmaCrossover', group: 'swing', defaultPct: 16 },
  { id: 'ema_trend_bias', query: 'weightEmaTrend', group: 'swing', defaultPct: 12 },
  { id: 'supertrend', query: 'weightSupertrend', group: 'swing', defaultPct: 13 },
  { id: 'rsi_recovery', query: 'weightRsiRecovery', group: 'swing', defaultPct: 13 },
  { id: 'macd_momentum', query: 'weightMacd', group: 'swing', defaultPct: 12 },
  { id: 'adx_trend_strength', query: 'weightAdx', group: 'swing', defaultPct: 12 },
  { id: 'accumulation', query: 'weightAccumulation', group: 'swing', defaultPct: 10 },
  { id: 'volume_breakout', query: 'weightVolumeBreakout', group: 'swing', defaultPct: 12 },
  { id: 'hunt_up', query: 'weightHuntUp', group: 'hunt', defaultPct: 0 },
  { id: 'forensics', query: 'weightForensics', group: 'tape', defaultPct: 0 },
  { id: 'similar_setup', query: 'weightSimilarSetup', group: 'tape', defaultPct: 0 },
] as const;

export type CustomSignalFactorId = (typeof CUSTOM_SIGNAL_FACTORS)[number]['id'];

export const CUSTOM_SIGNAL_BUILT_IN_ID = '';
