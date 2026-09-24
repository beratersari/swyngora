import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatCompactAmount, formatPrice } from '@/libs/utils';
import { columnPeak, profileLadderRows, profileVenue, splitPct } from './helpers';
import { Head, Legend, Row, Seg, Stack, Swatch, Track } from './VolumeProfileLadder.styles';
import type { VolumeProfileLadderProps } from './VolumeProfileLadder.types';
import type { VolumeProfileBin } from '@/libs/api';

function Meter({
  first,
  second,
  firstSide,
  secondSide,
  peak,
}: {
  first?: string;
  second?: string;
  firstSide: 'up' | 'down';
  secondSide: 'up' | 'down';
  peak: number;
}) {
  const pct = splitPct(first, second, peak);
  return (
    <div>
      <Track>
        <Seg $pct={pct.left} $side={firstSide} />
        <Seg $pct={pct.right} $side={secondSide} />
      </Track>
      <Text variant="caption" color="secondary">
        {`${formatCompactAmount(first)} / ${formatCompactAmount(second)}`}
      </Text>
    </div>
  );
}

function rowTitle(row: VolumeProfileBin): string {
  return [
    formatPrice(row.price),
    `vol ${formatCompactAmount(row.volume)}`,
    `oi +${formatCompactAmount(row.oiOpened)} / -${formatCompactAmount(row.oiClosed)}`,
    `liq L ${formatCompactAmount(row.longLiq)} S ${formatCompactAmount(row.shortLiq)}`,
  ].join(' · ');
}

export function VolumeProfileLadder({ report }: VolumeProfileLadderProps) {
  const { t } = useTranslation('detail');
  const venue = profileVenue(report);
  const rows = profileLadderRows(venue?.bins);
  if (!venue || rows.length === 0) return null;

  const volPeak = columnPeak(rows, 'buyVolume', 'sellVolume');
  const oiPeak = columnPeak(rows, 'oiOpened', 'oiClosed');
  const liqPeak = columnPeak(rows, 'longLiq', 'shortLiq');

  return (
    <Stack>
      <Legend>
        <Text variant="caption" color="secondary">
          <Swatch $side="up" />
          {t('flow.ladder.buy')}
        </Text>
        <Text variant="caption" color="secondary">
          <Swatch $side="down" />
          {t('flow.ladder.sell')}
        </Text>
        <Text variant="caption" color="secondary">
          <Swatch $side="up" />
          {t('flow.ladder.opened')}
        </Text>
        <Text variant="caption" color="secondary">
          <Swatch $side="down" />
          {t('flow.ladder.closed')}
        </Text>
        <Text variant="caption" color="secondary">
          <Swatch $side="down" />
          {t('flow.ladder.longLiq')}
        </Text>
        <Text variant="caption" color="secondary">
          <Swatch $side="up" />
          {t('flow.ladder.shortLiq')}
        </Text>
      </Legend>
      <Head>
        <Text variant="caption" color="secondary">
          {t('flow.ladder.price')}
        </Text>
        <Text variant="caption" color="secondary">
          {t('flow.ladder.volume')}
        </Text>
        <Text variant="caption" color="secondary">
          {t('flow.ladder.oi')}
        </Text>
        <Text variant="caption" color="secondary">
          {t('flow.ladder.liq')}
        </Text>
      </Head>
      <div role="list" aria-label={t('flow.ladder.aria')}>
        {rows.map((row) => (
          <Row key={`${row.price}-${row.high}`} $poc={row.isPoc} role="listitem" title={rowTitle(row)}>
            <Text variant="numeric" color="primary">
              {formatPrice(row.price)}
            </Text>
            <Meter first={row.buyVolume} second={row.sellVolume} firstSide="up" secondSide="down" peak={volPeak} />
            <Meter first={row.oiOpened} second={row.oiClosed} firstSide="up" secondSide="down" peak={oiPeak} />
            <Meter first={row.longLiq} second={row.shortLiq} firstSide="down" secondSide="up" peak={liqPeak} />
          </Row>
        ))}
      </div>
      <Text variant="caption" color="secondary">
        {t('flow.ladder.hint')}
      </Text>
    </Stack>
  );
}
