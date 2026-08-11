import { Chip, Rail } from './IntervalRail.styles';
import type { IntervalRailProps } from './IntervalRail.types';

/** Compact interval picker — pill rail instead of a Select. */
export function IntervalRail({
  intervals,
  value,
  onChange,
  loading = false,
  'aria-label': ariaLabel,
}: IntervalRailProps) {
  const options = intervals.length > 0 ? intervals : [value];
  return (
    <Rail role="radiogroup" aria-label={ariaLabel} aria-busy={loading || undefined}>
      {options.map((iv) => (
        <Chip
          key={iv}
          type="button"
          role="radio"
          aria-checked={iv === value}
          $active={iv === value}
          onClick={() => onChange(iv)}
        >
          {iv}
        </Chip>
      ))}
    </Rail>
  );
}
