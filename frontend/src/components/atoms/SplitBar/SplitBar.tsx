import { Fill, Track } from './SplitBar.styles';
import { splitWidths } from './helpers';
import type { SplitBarProps } from './SplitBar.types';

/** Two or three shares of one whole, such as bid versus ask or coins up versus down. */
export function SplitBar({ parts, ariaLabel }: SplitBarProps) {
  const widths = splitWidths(parts);
  return (
    <Track role="img" aria-label={ariaLabel}>
      {parts.map((part, index) => (
        <Fill key={part.key} $color={part.color} $width={widths[index] ?? '0%'} />
      ))}
    </Track>
  );
}
