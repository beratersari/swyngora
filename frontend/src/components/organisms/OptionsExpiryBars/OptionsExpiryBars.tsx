import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { formatCompactAmount } from '@/libs/utils';
import { callShare, expiryShare } from './helpers';
import { CallSlice, Fill, PutSlice, Row, Stack, Track } from './OptionsExpiryBars.styles';
import type { OptionsExpiryBarsProps } from './OptionsExpiryBars.types';

export function OptionsExpiryBars({ expiries, isLoading }: OptionsExpiryBarsProps) {
  const { t } = useTranslation('options');
  const rows = expiries ?? [];

  if (isLoading && rows.length === 0) {
    return (
      <Stack>
        {Array.from({ length: 5 }, (_, i) => (
          <Skeleton key={i} height={18} />
        ))}
      </Stack>
    );
  }

  if (rows.length === 0) {
    return (
      <Text variant="bodySm" color="secondary">
        {t('expiry.empty')}
      </Text>
    );
  }

  return (
    <Stack role="list" aria-label={t('expiry.aria')}>
      {rows.map((row) => {
        const key = row.expiry ?? row.label ?? '';
        const share = expiryShare(row);
        const calls = callShare(row);
        return (
          <Row key={key} role="listitem">
            <Text variant="caption" color="secondary">
              {row.label ?? '—'}
            </Text>
            <Track title={t('expiry.barTitle', { share: share.toFixed(1) })}>
              <Fill $width={Math.max(share, 2)}>
                <CallSlice $share={calls} />
                <PutSlice />
              </Fill>
            </Track>
            <Text variant="caption" color="primary">
              {formatCompactAmount(row.openInterestUsd)}
            </Text>
          </Row>
        );
      })}
    </Stack>
  );
}
