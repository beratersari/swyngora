export const LIQ_CARD_WINDOWS = ['1h', '4h', '12h', '24h'] as const;

export type LiqCardWindowId = (typeof LIQ_CARD_WINDOWS)[number];
