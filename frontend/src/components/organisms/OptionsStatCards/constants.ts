export const OPTIONS_STAT_KEYS = ['oi', 'volume', 'pcr', 'dvol', 'atm'] as const;

export type OptionsStatKey = (typeof OPTIONS_STAT_KEYS)[number];
