import { Spin } from 'antd';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { Skeleton } from '@/components/atoms/Skeleton';
import {
  asksHighToLow,
  columnWidths,
  formatBookAmount,
  formatBookPrice,
  formatGroupLabel,
  groupLabelDecimals,
  markdownBookRow,
  markdownRule,
  priceDecimalsFromGroup,
  qtyDecimalsFromLevels,
  sharedPriceExponent,
  visibleGroupSteps,
} from './helpers';
import {
  BookLine,
  BookPre,
  GroupField,
  GroupSelect,
  MetaRow,
  Panel,
  TitleRow,
} from './OrderBookPanel.styles';
import type { OrderBookPanelProps } from './OrderBookPanel.types';

function bookCells(
  price: string,
  qty: string,
  sum: string,
  wall: string | undefined,
): [string, string, string] {
  return [wall ? `${price} ${wall}` : price, qty, sum];
}

export function OrderBookPanel({
  book,
  group,
  onGroupChange,
  isLoading,
  isFetching,
  errorMessage,
}: OrderBookPanelProps) {
  const { t } = useTranslation('detail');
  const suggested = book?.suggestedGroupSizes?.length
    ? book.suggestedGroupSizes
    : group
      ? [group]
      : [];
  const activeGroup = group || book?.groupSize || suggested[0] || '';
  const steps = visibleGroupSteps(suggested, activeGroup);
  const selectValue = steps.includes(activeGroup) ? activeGroup : (steps[0] ?? '');
  const stepDecimals = groupLabelDecimals(steps);
  const asks = asksHighToLow(book?.asks);
  const bids = book?.bids ?? [];
  const priceDecimals = priceDecimalsFromGroup(selectValue || book?.groupSize);
  const priceExp = sharedPriceExponent(selectValue || book?.groupSize, book?.lastPrice);
  const qtyDecimals = qtyDecimalsFromLevels([...asks, ...bids]);
  const wallLabel = t('orderBook.wall');
  const { colWidths, headerRow, midRow, rule } = useMemo(() => {
    const header = [t('orderBook.price'), t('orderBook.size'), t('orderBook.sum')];
    const mid = [
      formatBookPrice(book?.lastPrice, priceDecimals, priceExp),
      t('orderBook.mid'),
      '',
    ];
    const body = [...asks, ...bids].map((lv) =>
      bookCells(
        formatBookPrice(lv.price, priceDecimals, priceExp),
        formatBookAmount(lv.quantity, qtyDecimals),
        formatBookAmount(lv.cumulative, qtyDecimals),
        lv.isWall ? wallLabel : undefined,
      ),
    );
    const widths = columnWidths([header, mid, ...body]);
    return {
      colWidths: widths,
      headerRow: markdownBookRow(header, widths),
      midRow: markdownBookRow(mid, widths),
      rule: markdownRule(widths),
    };
  }, [asks, bids, book?.lastPrice, priceDecimals, priceExp, qtyDecimals, t, wallLabel]);
  const imbalance = book?.imbalance ?? 0;
  const imbalanceLabel =
    imbalance > 0.08 ? t('orderBook.bidHeavy') : imbalance < -0.08 ? t('orderBook.askHeavy') : t('orderBook.balanced');

  return (
    <Panel data-testid="order-book">
      <TitleRow>
        <Text variant="h4" color="primary">
          {t('orderBook.title')}
        </Text>
        {isFetching && !isLoading ? <Spin size="small" /> : null}
      </TitleRow>
      <Text variant="caption" color="secondary">
        {t('orderBook.subtitle')}
      </Text>

      {steps.length > 0 ? (
        <GroupField>
          <Text variant="caption" color="secondary" as="label" id="order-book-group-label">
            {t('orderBook.group')}
          </Text>
          <GroupSelect
            id="order-book-group"
            aria-labelledby="order-book-group-label"
            value={selectValue}
            onChange={(event) => onGroupChange(event.target.value)}
          >
            {steps.map((step) => (
              <option key={step} value={step}>
                {formatGroupLabel(step, stepDecimals)}
              </option>
            ))}
          </GroupSelect>
        </GroupField>
      ) : null}

      {errorMessage ? (
        <Text variant="body" color="secondary">
          {errorMessage}
        </Text>
      ) : isLoading && !book ? (
        <Skeleton height={280} />
      ) : !book || (bids.length === 0 && asks.length === 0) ? (
        <Text variant="body" color="secondary">
          {t('orderBook.empty')}
        </Text>
      ) : (
        <>
          <MetaRow>
            <Text variant="caption" color="secondary">
              {t('orderBook.spread')}: {book.spread || '—'}
              {book.spreadPct ? ` (${book.spreadPct}%)` : ''}
            </Text>
            <Text variant="caption" color="secondary">
              {imbalanceLabel}
            </Text>
          </MetaRow>
          <BookPre>
            <BookLine $mid>{headerRow}</BookLine>
            <BookLine $mid>{rule}</BookLine>
            {asks.map((lv) => (
              <BookLine key={`a-${lv.price}`} $side="ask" data-testid={`ob-ask-${lv.price}`}>
                {markdownBookRow(
                  bookCells(
                    formatBookPrice(lv.price, priceDecimals, priceExp),
                    formatBookAmount(lv.quantity, qtyDecimals),
                    formatBookAmount(lv.cumulative, qtyDecimals),
                    lv.isWall ? wallLabel : undefined,
                  ),
                  colWidths,
                )}
              </BookLine>
            ))}
            <BookLine $mid>{midRow}</BookLine>
            {bids.map((lv) => (
              <BookLine key={`b-${lv.price}`} $side="bid" data-testid={`ob-bid-${lv.price}`}>
                {markdownBookRow(
                  bookCells(
                    formatBookPrice(lv.price, priceDecimals, priceExp),
                    formatBookAmount(lv.quantity, qtyDecimals),
                    formatBookAmount(lv.cumulative, qtyDecimals),
                    lv.isWall ? wallLabel : undefined,
                  ),
                  colWidths,
                )}
              </BookLine>
            ))}
          </BookPre>
        </>
      )}
    </Panel>
  );
}
