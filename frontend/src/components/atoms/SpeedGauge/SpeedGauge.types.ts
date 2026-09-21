export type SpeedGaugeZone = {
  from: number;
  to: number;
  color: string;
};

export type SpeedGaugeProps = {
  /** 0 sits on the left, 100 on the right. */
  value?: number | null;
  minLabel: string;
  maxLabel: string;
  ariaLabel: string;
};
