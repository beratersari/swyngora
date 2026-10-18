import { Button } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatPrice } from '@/libs/utils/formatPrice';
import type { SpreadOrderListProps } from './SpreadOrderList.types';

/**
 * Open spread orders on the paper desk. Cancel stops slices that have not run.
 */
export function SpreadOrderList({ items, canceling = false, onCancel }: SpreadOrderListProps) {
  const { t } = useTranslation('portfolio');
  if (items.length === 0) return null;
  return (
    <div>
      <Text variant="caption" color="primary">
        {t('orders.spreadTitle')}
      </Text>
      {items.map((order) => (
        <div key={order.id} data-testid="spread-order">
          <Text variant="caption" color="secondary">
            {t('orders.spreadRow', {
              side: order.side,
              symbol: order.symbol,
              filled: formatPrice(order.filledQuantity),
              quantity: formatPrice(order.quantity),
              done: order.slicesDone,
              count: order.sliceCount,
              minutes: order.durationMinutes,
            })}
          </Text>
          <Button
            size="small"
            disabled={canceling}
            aria-label={t('orders.spreadCancel')}
            onClick={() => onCancel(order.id)}
          >
            {t('orders.spreadCancel')}
          </Button>
        </div>
      ))}
    </div>
  );
}
