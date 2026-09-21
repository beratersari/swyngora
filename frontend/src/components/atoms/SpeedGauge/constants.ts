import type { SpeedGaugeZone } from './SpeedGauge.types';

export const GAUGE_CX = 80;
export const GAUGE_CY = 78;
export const GAUGE_R = 58;

/** Fear on the left, greed on the right. The same sweep reads down/ask vs up/bid. */
export const SPEED_GAUGE_ZONES: SpeedGaugeZone[] = [
  { from: 0, to: 25, color: '#EA3943' },
  { from: 25, to: 45, color: '#F5B544' },
  { from: 45, to: 55, color: '#C7CDD8' },
  { from: 55, to: 75, color: '#5DCAA5' },
  { from: 75, to: 100, color: '#16C784' },
];
