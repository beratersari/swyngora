import { Alert, InputNumber } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatPrice } from '@/libs/utils/formatPrice';
import { EXECUTION_STYLES, type ExecutionStyleId } from './constants';
import { Card, Columns, Line, Panel, Pick, Schedule } from './ExecutionCompare.styles';
import type { ExecutionCompareProps } from './ExecutionCompare.types';
import { cheapestStyles, formatComparePct, legVenues, unfilledReason } from './helpers';

/**
 * Fee and slippage for one size, three ways. Choosing a column does not place the order.
 * The page owns the request.
 */
export function ExecutionCompare({
  compare,
  hasSize = false,
  loading = false,
  failed = false,
  selected,
  minutes,
  slices,
  onSelect,
  onMinutes,
  onSlices,
}: ExecutionCompareProps) {
  const { t } = useTranslation('portfolio');
  const rows = compare?.rows ?? [];
  const cheap = cheapestStyles(compare?.side, rows);
  const sell = compare?.side === 'sell';

  return (
    <Panel role="region" aria-label={t('compare.title')}>
      <Text variant="caption" color="primary">
        {t('compare.title')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('compare.note')}
      </Text>
      {loading && rows.length === 0 ? (
        <Text variant="caption" color="secondary">
          {t('compare.pricing')}
        </Text>
      ) : null}
      {failed && rows.length === 0 ? <Alert type="warning" showIcon message={t('compare.failed')} /> : null}
      {!hasSize && !loading && !failed && rows.length === 0 ? (
        <Text variant="caption" color="secondary">
          {t('compare.enterSize')}
        </Text>
      ) : null}
      {compare && (compare.touchPrice ?? 0) > 0 ? (
        <Text variant="caption" color="secondary">
          {t('compare.touch', {
            price: formatPrice(compare.touchPrice),
            venue: compare.touchExchange || '—',
          })}
        </Text>
      ) : null}
      <Columns>
          {EXECUTION_STYLES.map((style) => {
            const row = rows.find((item) => item.style === style);
            const filled = row?.status === 'filled';
            const cash = sell ? row?.cashCredit : row?.cashDebit;
            const venues = row ? legVenues(row) : '';
            return (
              <Card key={style} $selected={selected === style}>
                <Pick
                  type="button"
                  aria-pressed={selected === style}
                  onClick={() => onSelect(style)}
                >
                  <Text variant="caption" color="primary">
                    {t(`compare.styles.${style}`)}
                  </Text>
                  {cheap.has(style) ? (
                    <Text variant="caption" color="secondary">
                      {sell ? t('compare.lowestSell') : t('compare.lowestBuy')}
                    </Text>
                  ) : null}
                  {row && !filled ? (
                    <Text variant="caption" color="secondary">
                      {unfilledReason(t, row.status, row.schedule)}
                    </Text>
                  ) : null}
                  {filled ? (
                    <>
                      <Line>
                        <Text variant="caption" color="secondary">
                          {t('compare.slippage')}
                        </Text>
                        <Text variant="caption" color="primary">
                          {formatComparePct(row?.slippagePct)}
                        </Text>
                      </Line>
                      <Line>
                        <Text variant="caption" color="secondary">
                          {t('compare.fee')}
                        </Text>
                        <Text variant="caption" color="primary">
                          {formatPrice(row?.fee)} · {formatComparePct(row?.feePct)}
                        </Text>
                      </Line>
                      <Line>
                        <Text variant="caption" color="secondary">
                          {t('compare.average')}
                        </Text>
                        <Text variant="caption" color="primary">
                          {formatPrice(row?.averagePrice)}
                        </Text>
                      </Line>
                      <Line>
                        <Text variant="caption" color="secondary">
                          {sell ? t('compare.credit') : t('compare.debit')}
                        </Text>
                        <Text variant="caption" color="primary">
                          {formatPrice(cash)}
                        </Text>
                      </Line>
                      {row?.savedVsMarket != null ? (
                        <Line>
                          <Text variant="caption" color="secondary">
                            {t('compare.saved')}
                          </Text>
                          <Text variant="caption" color="primary">
                            {formatPrice(row.savedVsMarket)}
                          </Text>
                        </Line>
                      ) : null}
                      {venues ? (
                        <Text variant="caption" color="secondary">
                          {venues}
                        </Text>
                      ) : null}
                    </>
                  ) : null}
                  {row?.missing && row.missing.length > 0 ? (
                    <Text variant="caption" color="secondary">
                      {t('compare.missing', { names: row.missing.join(', ') })}
                    </Text>
                  ) : null}
                  {style === 'spread' && row?.assumption === 'refill' ? (
                    <Text variant="caption" color="secondary">
                      {t('compare.refill')}
                    </Text>
                  ) : null}
                </Pick>
                {style === 'spread' ? (
                  <Schedule>
                    <InputNumber
                      min={1}
                      max={1440}
                      step={1}
                      precision={0}
                      value={minutes}
                      onChange={(value) => onMinutes(typeof value === 'number' ? value : null)}
                      style={{ minWidth: 110 }}
                      aria-label={t('trade.spreadMinutes')}
                      placeholder={t('trade.spreadMinutes')}
                    />
                    <InputNumber
                      min={2}
                      max={60}
                      step={1}
                      precision={0}
                      value={slices}
                      onChange={(value) => onSlices(typeof value === 'number' ? value : null)}
                      style={{ minWidth: 110 }}
                      aria-label={t('trade.spreadSlices')}
                      placeholder={t('trade.spreadSlices')}
                    />
                  </Schedule>
                ) : null}
              </Card>
            );
          })}
        </Columns>
    </Panel>
  );
}

export type { ExecutionStyleId };
