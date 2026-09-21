import { SpeedGauge } from '@/components/atoms/SpeedGauge';
import { Text } from '@/components/atoms/Text';
import { Card, Copy } from './DeskGaugeCard.styles';
import type { DeskGaugeCardProps } from './DeskGaugeCard.types';

/** Compact desk read: a speedometer beside one number. The card opens the detail dialog. */
export function DeskGaugeCard({
  title,
  value,
  caption,
  gaugeValue,
  minLabel,
  maxLabel,
  isLoading = false,
  onClick,
}: DeskGaugeCardProps) {
  return (
    <Card type="button" onClick={onClick} aria-haspopup="dialog">
      <SpeedGauge
        value={gaugeValue}
        minLabel={minLabel}
        maxLabel={maxLabel}
        ariaLabel={gaugeValue == null ? title : `${title} ${Math.round(gaugeValue)}`}
      />
      <Copy>
        <Text variant="label" color="primary">
          {title}
        </Text>
        <Text variant="h3" color="primary" isLoading={isLoading} skeletonWidth={48}>
          {value}
        </Text>
        {caption ? (
          <Text variant="caption" color="secondary">
            {caption}
          </Text>
        ) : null}
      </Copy>
    </Card>
  );
}
