import { useState } from 'react';
import { Segmented } from 'antd';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { formatCompactAmount, formatPrice } from '@/libs/utils';
import { changeBarPct, maxAbsChange, signedChange, windowRows } from './helpers';

function formatSignedMove(raw: string | undefined): string {
  const n = signedChange(raw);
  const body = formatCompactAmount(n);
  if (n > 0) return `+${body}`;
  return body;
}
import { Fill, Head, List, Row, Side, Stack, Track } from './OptionsStrikeOI.styles';
import type { OptionsStrikeOIProps } from './OptionsStrikeOI.types';

const WINDOWS = ['1h', '4h', '24h', '7d'] as const;

export function OptionsStrikeOI({ windows, expiry, isLoading }: OptionsStrikeOIProps) {
  const { t } = useTranslation('options');
  const [windowId, setWindowId] = useState<(typeof WINDOWS)[number]>('24h');
  const rows = windowRows(windows, windowId, expiry);
  const peak = maxAbsChange(rows);
  const selected = (windows ?? []).find((row) => row.window === windowId);
  const collecting = !selected?.complete;

  if (isLoading && !windows) {
    return <Skeleton height={160} />;
  }

  return (
    <Stack>
      <Head>
        <Text variant="caption" color="secondary">
          {t('strikeOi.legend')}
        </Text>
        <Segmented
          size="small"
          value={windowId}
          options={WINDOWS.map((value) => ({ value, label: t(`history.window.${value}`) }))}
          onChange={(next) => setWindowId(next as (typeof WINDOWS)[number])}
          aria-label={t('strikeOi.windowAria')}
        />
      </Head>
      {collecting ? (
        <Text variant="bodySm" color="secondary">
          {t('strikeOi.collecting')}
        </Text>
      ) : rows.length === 0 ? (
        <Text variant="bodySm" color="secondary">
          {t('strikeOi.empty')}
        </Text>
      ) : (
        <List aria-label={t('strikeOi.aria')}>
          {rows.map((row) => {
            const call = signedChange(row.callChange);
            const put = signedChange(row.putChange);
            return (
              <Row key={`${row.expiry}-${row.strike}`}>
                <Text variant="caption" color="secondary">
                  {formatPrice(row.strike)}
                </Text>
                <Side title={t('strikeOi.callTitle', { now: row.callOi ?? '0', past: row.callPast ?? '0' })}>
                  <Text variant="caption" color={call > 0 ? 'success' : call < 0 ? 'error' : 'secondary'}>
                    {t('strikeOi.call', { value: formatSignedMove(row.callChange) })}
                  </Text>
                  <Track>
                    <Fill $width={changeBarPct(row.callChange, peak)} $up={call >= 0} />
                  </Track>
                </Side>
                <Side title={t('strikeOi.putTitle', { now: row.putOi ?? '0', past: row.putPast ?? '0' })}>
                  <Text variant="caption" color={put > 0 ? 'success' : put < 0 ? 'error' : 'secondary'}>
                    {t('strikeOi.put', { value: formatSignedMove(row.putChange) })}
                  </Text>
                  <Track>
                    <Fill $width={changeBarPct(row.putChange, peak)} $up={put >= 0} />
                  </Track>
                </Side>
              </Row>
            );
          })}
        </List>
      )}
    </Stack>
  );
}
