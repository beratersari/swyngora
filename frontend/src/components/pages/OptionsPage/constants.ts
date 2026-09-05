export const OPTIONS_ASSETS = ['BTC', 'ETH'] as const;

export type OptionsAsset = (typeof OPTIONS_ASSETS)[number];

export const OPTIONS_POLL_MS = 30_000;
