import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { formatCompactAmount, formatPrice } from '@/libs/utils';
import { formatMovePct, formatSignedCompact, maxPayout, payoutPct } from './helpers';
import { Card, Cards, Fill, Stack, StrikeRow, Track } from './OptionsExpiryDetail.styles';
import type { OptionsExpiryDetailProps } from './OptionsExpiryDetail.types';

export function OptionsExpiryDetail({ expiry, indexPrice, isLoading }: OptionsExpiryDetailProps) {
  const { t } = useTranslation('options');
  const strikes = expiry?.strikes ?? [];
  const peak = maxPayout(strikes);

  if (isLoading && !expiry) {
    return <Skeleton height={160} />;
  }
  if (!expiry) {
    return (
      <Text variant="bodySm" color="secondary">
        {t('detail.empty')}
      </Text>
    );
  }

  return (
    <Stack>
      <Cards>
        <Card>
          <Text variant="caption" color="secondary">
            {t('detail.move')}
          </Text>
          <Text variant="h3" color="primary">
            {formatMovePct(expiry.expectedMovePct)}
          </Text>
          <Text variant="caption" color="secondary">
            {expiry.expectedMoveAbs
              ? t('detail.moveAbs', { value: formatPrice(expiry.expectedMoveAbs) })
              : t('detail.moveEmpty')}
          </Text>
        </Card>
        <Card>
          <Text variant="caption" color="secondary">
            {t('detail.range')}
          </Text>
          <Text variant="h3" color="primary">
            {expiry.expectedLow && expiry.expectedHigh
              ? `${formatPrice(expiry.expectedLow)} – ${formatPrice(expiry.expectedHigh)}`
              : '—'}
          </Text>
          <Text variant="caption" color="secondary">
            {t('detail.rangeHint', { days: expiry.daysToExpiry ?? 0 })}
          </Text>
        </Card>
        <Card>
          <Text variant="caption" color="secondary">
            {t('detail.maxPain')}
          </Text>
          <Text variant="h3" color="primary">
            {expiry.maxPain ? formatPrice(expiry.maxPain) : '—'}
          </Text>
          <Text variant="caption" color="secondary">
            {indexPrice
              ? t('detail.maxPainHint', { spot: formatPrice(indexPrice) })
              : t('detail.maxPainEmpty')}
          </Text>
        </Card>
        <Card>
          <Text variant="caption" color="secondary">
            {t('detail.gex')}
          </Text>
          <Text variant="h3" color="primary">
            {expiry.gammaNetUsd ? formatSignedCompact(expiry.gammaNetUsd) : '—'}
          </Text>
          <Text variant="caption" color="secondary">
            {expiry.gammaRegime
              ? t('detail.gexHint', {
                  regime: t(`gamma.regimeLabel.${expiry.gammaRegime}` as 'gamma.regimeLabel.flat'),
                })
              : t('detail.gexEmpty')}
          </Text>
        </Card>
        <Card>
          <Text variant="caption" color="secondary">
            {t('detail.flip')}
          </Text>
          <Text variant="h3" color="primary">
            {expiry.gammaFlip ? formatPrice(expiry.gammaFlip) : '—'}
          </Text>
          <Text variant="caption" color="secondary">
            {indexPrice ? t('detail.flipHint', { spot: formatPrice(indexPrice) }) : t('detail.flipEmpty')}
          </Text>
        </Card>
      </Cards>

      {strikes.length > 0 ? (
        <div>
          <Text variant="caption" color="secondary">
            {t('detail.payoutLegend')}
          </Text>
          {strikes.map((s) => (
            <StrikeRow key={s.strike} $maxPain={s.isMaxPain}>
              <Text variant="caption" color="secondary">
                {formatPrice(s.strike)}
              </Text>
              <Track title={formatCompactAmount(s.payout)}>
                <Fill $width={Math.max(payoutPct(s.payout, peak), s.isMaxPain ? 2 : 0)} />
              </Track>
              <Text variant="caption" color="primary">
                {s.isMaxPain ? t('detail.painMark') : formatCompactAmount(s.payout)}
              </Text>
            </StrikeRow>
          ))}
        </div>
      ) : null}
    </Stack>
  );
}
