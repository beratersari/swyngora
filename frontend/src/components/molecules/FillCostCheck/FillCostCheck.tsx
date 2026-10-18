import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatPrice } from '@/libs/utils/formatPrice';
import type { FillCostCheckView } from './FillCostCheck.types';
import { signedQuote } from './helpers';

const SIZE_EPSILON = 1e-8;

/**
 * The fee and slippage quoted before the order, beside what the fills paid.
 * The page passes the check from the finished trade.
 */
export function FillCostCheck({ check }: { check: FillCostCheckView }) {
  const { t } = useTranslation('portfolio');
  const filled = check.filledQuantity ?? 0;
  const quoted = check.quotedQuantity ?? 0;
  const partial = Math.abs(filled - quoted) > SIZE_EPSILON;
  return (
    <div data-testid="fill-cost-check">
      <Text variant="caption">
        {t('trades.costFee', {
          actual: formatPrice(check.actualFee),
          expected: formatPrice(check.expectedFee),
          delta: signedQuote(check.feeDelta),
        })}
      </Text>
      <Text variant="caption">
        {t('trades.costSlip', {
          actual: formatPrice(check.actualSlippage),
          expected: formatPrice(check.expectedSlippage),
          delta: signedQuote(check.slippageDelta),
        })}
      </Text>
      {partial ? (
        <Text variant="caption" color="secondary">
          {t('trades.costFilled', {
            filled: formatPrice(filled),
            quoted: formatPrice(quoted),
          })}
        </Text>
      ) : null}
    </div>
  );
}
