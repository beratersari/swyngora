import { Select } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import {
  LIQ_HEATMAP_RANGES,
} from '@/components/organisms/LiquidationHeatmap';
import { formatLeverageMix } from '@/components/organisms/LiquidationHeatmap/LiquidationHeatmap.helpers';
import { formatCompactUsd, formatDateTime, formatPrice } from '@/libs/utils';
import {
  formatSignedCompactUsd,
  pickOpenLevelBook,
  openLevelTableRows,
} from './OpenLiquidationLevels.helpers';
import { Panel, Summary, Table, TableWrap, Td, Th } from './OpenLiquidationLevels.styles';
import type { OpenLiquidationLevelsProps } from './OpenLiquidationLevels.types';

export function OpenLiquidationLevels({
  data,
  venue,
  range,
  onRangeChange,
  isLoading = false,
  errorMessage,
}: OpenLiquidationLevelsProps) {
  const { t } = useTranslation(['liquidations', 'detail', 'common']);
  const book = pickOpenLevelBook(data, venue);
  const rows = openLevelTableRows(book);
  const mix = formatLeverageMix(data?.leverageMix);
  const missing = (data?.missingVenues ?? []).filter((id) => id.length > 0);
  const withheld = book != null && book.definite === false;

  return (
    <Panel data-testid="open-liquidation-levels">
      <Text variant="bodySm" color="secondary">
        {t('liquidations:open.subtitle')}
      </Text>
      {onRangeChange ? (
        <div>
          <Text variant="caption" color="secondary" id="open-level-range-label">
            {t('detail:liqHeatmap.range')}
          </Text>
          <Select
            aria-labelledby="open-level-range-label"
            size="small"
            value={range}
            style={{ minWidth: 120, display: 'block', marginTop: 4 }}
            options={LIQ_HEATMAP_RANGES.map((id) => ({
              value: id,
              label: t(`detail:liqHeatmap.ranges.${id}`),
            }))}
            onChange={onRangeChange}
          />
        </div>
      ) : null}
      {mix ? (
        <Text variant="caption" color="secondary" data-testid="open-level-mix">
          {t('liquidations:open.mix', { mix })}
        </Text>
      ) : null}
      {missing.length > 0 ? (
        <Text variant="caption" color="secondary" data-testid="open-level-missing">
          {t('liquidations:open.missing', {
            venues: missing
              .map((id) => t(`common:exchanges.${id}`, { defaultValue: id }))
              .join(', '),
          })}
        </Text>
      ) : null}
      {withheld ? (
        <Text variant="bodySm" color="secondary" data-testid="open-level-withheld">
          {t('liquidations:open.withheld')}
        </Text>
      ) : null}
      {errorMessage ? (
        <Text variant="bodySm" color="secondary">
          {errorMessage}
        </Text>
      ) : null}
      {isLoading && !book ? (
        <Text variant="bodySm" color="secondary">
          {t('liquidations:open.loading')}
        </Text>
      ) : null}
      {book?.definite ? (
        <>
          <Summary>
            <Text variant="bodySm" data-testid="open-level-long">
              {t('liquidations:open.long')}: {formatCompactUsd(book.openLong)}
            </Text>
            <Text variant="bodySm" data-testid="open-level-short">
              {t('liquidations:open.short')}: {formatCompactUsd(book.openShort)}
            </Text>
            <Text variant="bodySm" data-testid="open-level-delta">
              {t('liquidations:open.delta')}: {formatCompactUsd(book.delta)}
            </Text>
            <Text variant="bodySm" data-testid="open-level-hits">
              {t('liquidations:open.hits')}: {book.hitLots ?? 0}
            </Text>
            <Text variant="bodySm" data-testid="open-level-closed">
              {t('liquidations:open.closed')}: {book.closedLots ?? 0}
            </Text>
          </Summary>
          {(['day', '4h'] as const).map((id) => {
            const row = book.anchors?.find((anchor) => anchor.anchor === id);
            if (!row) return null;
            if (!row.complete) {
              return (
                <Text
                  key={id}
                  variant="caption"
                  color="secondary"
                  data-testid={`open-level-anchor-${id}`}
                >
                  {t(id === 'day' ? 'detail:openLevels.dayMissing' : 'detail:openLevels.blockMissing')}
                </Text>
              );
            }
            return (
              <Text
                key={id}
                variant="caption"
                color="secondary"
                data-testid={`open-level-anchor-${id}`}
              >
                {t(id === 'day' ? 'detail:openLevels.sinceDay' : 'detail:openLevels.since4h', {
                  long: formatSignedCompactUsd(row.longChange),
                  short: formatSignedCompactUsd(row.shortChange),
                  delta: formatSignedCompactUsd(row.deltaChange),
                })}
              </Text>
            );
          })}
          {rows.length === 0 ? (
            <Text variant="bodySm" color="secondary">
              {t('liquidations:open.empty')}
            </Text>
          ) : (
            <TableWrap>
              <Table>
                <thead>
                  <tr>
                    <Th>{t('liquidations:open.side')}</Th>
                    <Th>{t('liquidations:open.leverage')}</Th>
                    <Th>{t('liquidations:open.price')}</Th>
                    <Th>{t('liquidations:open.notional')}</Th>
                    <Th>{t('liquidations:open.opened')}</Th>
                    <Th>{t('liquidations:open.exchange')}</Th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row) => (
                    <tr
                      key={`${row.exchange}-${row.side}-${row.leverage}-${row.price}-${row.openedAt}`}
                      data-testid="open-level-row"
                    >
                      <Td>{t(`liquidations:open.sides.${row.side}`)}</Td>
                      <Td>{row.leverage > 0 ? `${row.leverage}×` : '—'}</Td>
                      <Td>{formatPrice(row.price)}</Td>
                      <Td>{formatCompactUsd(row.notional)}</Td>
                      <Td>{formatDateTime(row.openedAt)}</Td>
                      <Td>{t(`common:exchanges.${row.exchange}`, { defaultValue: row.exchange })}</Td>
                    </tr>
                  ))}
                </tbody>
              </Table>
            </TableWrap>
          )}
        </>
      ) : null}
    </Panel>
  );
}
