export const EMERGENCY_KINDS = [
  'cancel_order',
  'cancel_spread',
  'pause_recurring',
  'close_spot',
  'close_margin',
  'cancel_pending',
] as const;

export type EmergencyKind = (typeof EMERGENCY_KINDS)[number];
