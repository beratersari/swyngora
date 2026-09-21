import { SPEED_GAUGE_ZONES, GAUGE_CX, GAUGE_CY, GAUGE_R } from './constants';
import { gaugeArc, gaugePoint } from './helpers';
import type { SpeedGaugeProps } from './SpeedGauge.types';

export function SpeedGauge({ value, minLabel, maxLabel, ariaLabel }: SpeedGaugeProps) {
  const known = value != null && Number.isFinite(value);
  const needle = known ? gaugePoint(value, GAUGE_R - 12) : null;

  return (
    <svg viewBox="0 0 160 108" width="148" height="100" role="img" aria-label={ariaLabel}>
      <path d={gaugeArc(0, 100, GAUGE_R)} fill="none" stroke="#EFF2F5" strokeWidth="10" strokeLinecap="butt" />
      {SPEED_GAUGE_ZONES.map((zone) => (
        <path
          key={`${zone.from}-${zone.to}`}
          d={gaugeArc(zone.from, zone.to)}
          fill="none"
          stroke={zone.color}
          strokeWidth="8"
          strokeLinecap="butt"
        />
      ))}
      {needle ? (
        <line
          x1={GAUGE_CX}
          y1={GAUGE_CY}
          x2={needle.x}
          y2={needle.y}
          stroke="#0D1421"
          strokeWidth="2.5"
          strokeLinecap="round"
        />
      ) : null}
      <circle cx={GAUGE_CX} cy={GAUGE_CY} r="4.5" fill="#0D1421" />
      <text x="8" y="100" fill="#616E85" fontSize="11" fontFamily="inherit">
        {minLabel}
      </text>
      <text x="152" y="100" fill="#616E85" fontSize="11" fontFamily="inherit" textAnchor="end">
        {maxLabel}
      </text>
    </svg>
  );
}
