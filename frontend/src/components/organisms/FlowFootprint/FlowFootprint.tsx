import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatCompactAmount, formatPrice } from '@/libs/utils';
import {
  barPct,
  footprintRows,
  footprintVenue,
  flowNum,
  imbalanceRatio,
  imbalanceSide,
  sidePeak,
} from './helpers';
import {
  Badge,
  Head,
  Legend,
  PriceCell,
  Row,
  Scroller,
  Seg,
  Side,
  Stack,
  Swatch,
  Track,
  TrackEnd,
} from './FlowFootprint.styles';
import type { FlowFootprintProps } from './FlowFootprint.types';
import type { FlowAtPricesLevel } from '@/libs/api';

function signedAmount(raw?: string): string {
  const n = flowNum(raw);
  const body = formatCompactAmount(Math.abs(n));
  if (n > 0) return `+${body}`;
  if (n < 0) return `-${body}`;
  return body;
}

function rowTitle(row: FlowAtPricesLevel): string {
  return [
    formatPrice(row.price),
    `buy ${formatCompactAmount(row.buyVolume)}`,
    `sell ${formatCompactAmount(row.sellVolume)}`,
    `delta ${signedAmount(row.delta)}`,
    `${row.trades ?? 0} prints`,
  ].join(' · ');
}

export function FlowFootprint({ report }: FlowFootprintProps) {
  const { t } = useTranslation('detail');
  const venue = footprintVenue(report);
  const rows = footprintRows(venue?.levels);
  if (!venue || rows.length === 0) return null;

  const peak = sidePeak(rows);
  const imbalances = rows.filter((row) => imbalanceSide(row));
  const buyImbalances = imbalances.filter((row) => imbalanceSide(row) === 'buy').length;
  const sellImbalances = imbalances.length - buyImbalances;

  return (
    <Stack>
      <Legend>
        <Text variant="caption" color="secondary">
          <Swatch $side="down" />
          {t('flow.footprint.sell')}
        </Text>
        <Text variant="caption" color="secondary">
          <Swatch $side="up" />
          {t('flow.footprint.buy')}
        </Text>
        <Text variant="caption" color="secondary">
          <Swatch $side="poc" />
          {t('flow.footprint.poc')}
        </Text>
        <Text variant="caption" color="secondary">
          {t('flow.footprint.imbalanceCount', { buy: buyImbalances, sell: sellImbalances })}
        </Text>
      </Legend>
      <Head>
        <Text variant="caption" color="secondary">
          {t('flow.footprint.sell')}
        </Text>
        <Text variant="caption" color="secondary">
          {t('flow.footprint.price')}
        </Text>
        <Text variant="caption" color="secondary">
          {t('flow.footprint.buy')}
        </Text>
        <Text variant="caption" color="secondary">
          {t('flow.footprint.delta')}
        </Text>
      </Head>
      <Scroller>
        <div role="list" aria-label={t('flow.footprint.aria')}>
          {rows.map((row) => {
            const side = imbalanceSide(row);
            const ratio = imbalanceRatio(row);
            const ratioLabel = Number.isFinite(ratio) ? `${ratio.toFixed(1)}×` : t('flow.footprint.oneSided');
            return (
              <Row
                key={`${row.price}-${row.high}`}
                $poc={row.isPoc}
                $side={side}
                role="listitem"
                title={rowTitle(row)}
              >
                <Side $align="end">
                  <Text variant="caption" color="secondary">
                    {formatCompactAmount(row.sellVolume)}
                  </Text>
                  <TrackEnd>
                    <Seg $pct={barPct(row.sellVolume, peak)} $side="down" />
                  </TrackEnd>
                </Side>
                <PriceCell>
                  <Text variant="numeric" color="primary">
                    {formatPrice(row.price)}
                  </Text>
                  {row.isPoc ? <Badge $kind="poc">{t('flow.footprint.poc')}</Badge> : null}
                  {side ? (
                    <Badge $kind={side}>
                      {t(side === 'buy' ? 'flow.footprint.buyImbalance' : 'flow.footprint.sellImbalance', {
                        ratio: ratioLabel,
                      })}
                    </Badge>
                  ) : null}
                </PriceCell>
                <Side $align="start">
                  <Track>
                    <Seg $pct={barPct(row.buyVolume, peak)} $side="up" />
                  </Track>
                  <Text variant="caption" color="secondary">
                    {formatCompactAmount(row.buyVolume)}
                  </Text>
                </Side>
                <Text variant="numeric" color={flowNum(row.delta) >= 0 ? 'success' : 'error'}>
                  {signedAmount(row.delta)}
                </Text>
              </Row>
            );
          })}
        </div>
      </Scroller>
      <Text variant="caption" color="secondary">
        {venue.truncated ? t('flow.footprint.truncated') : t('flow.footprint.hint')}
      </Text>
    </Stack>
  );
}
