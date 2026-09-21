import { Text } from '@/components/atoms/Text';
import { Card } from './DeskReadCard.styles';
import type { DeskReadCardProps } from './DeskReadCard.types';

/** Compact desk read: a title, one picture of the mix, and a short line. Opens the detail dialog. */
export function DeskReadCard({ title, value, caption, visual, isLoading = false, onClick }: DeskReadCardProps) {
  return (
    <Card type="button" onClick={onClick} aria-haspopup="dialog">
      <Text variant="label" color="primary">
        {title}
      </Text>
      {visual}
      <Text variant="h3" color="primary" isLoading={isLoading} skeletonWidth={72}>
        {value}
      </Text>
      {caption ? (
        <Text variant="caption" color="secondary">
          {caption}
        </Text>
      ) : null}
    </Card>
  );
}
