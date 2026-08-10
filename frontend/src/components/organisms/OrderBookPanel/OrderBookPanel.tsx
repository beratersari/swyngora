import { Segmented, Spin } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { Skeleton } from '@/components/atoms/Skeleton';
import type { OrderBookLevel } from '@/libs/api';
import { asksHighToLow, depthPct, maxNotional } from './helpers';
import {
  DepthBar,
  Head,
  Ladder,
  MetaRow,
  Panel,
  Price,
  Qty,
  Row,
  SpreadRow,
  TitleRow,
  WallTag,
} from './OrderBookPanel.styles';
import type { OrderBookPanelProps } from './OrderBookPanel.types';

function LevelRow({
  level,
  side,
  max,
}: {
  level: OrderBookLevel;
  side: 'bid' | 'ask';
  max: number;
}) {
  const { t } = useTranslation('detail');
  const pct = depthPct(level.notional, max);
  return (
    <Row type="button" $side={side} $wall={level.isWall} data-testid={`ob-${side}-${level.price}`}>
      <DepthBar $side={side} $pct={pct} />
      <Price $side={side}>
        {level.price}
        {level.isWall ? <WallTag>{t('orderBook.wall')}</WallTag> : null}
      </Price>
      <Qty>{level.quantity}</Qty>
      <Qty>{level.cumulative}</Qty>
    </Row>
  );
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
  const steps = book?.suggestedGroupSizes?.length
    ? book.suggestedGroupSizes
    : group
      ? [group]
      : [];
  const activeGroup = group || book?.groupSize || steps[0] || '';
  const asks = asksHighToLow(book?.asks);
  const bids = book?.bids ?? [];
  const max = Math.max(maxNotional(book?.asks), maxNotional(book?.bids));
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
        <Segmented
          size="small"
          value={activeGroup}
          options={steps.map((s) => ({ label: s, value: s }))}
          onChange={(v) => onGroupChange(String(v))}
          aria-label={t('orderBook.group')}
        />
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
          <Ladder>
            <Head>
              <span>{t('orderBook.price')}</span>
              <span style={{ textAlign: 'right' }}>{t('orderBook.size')}</span>
              <span style={{ textAlign: 'right' }}>{t('orderBook.sum')}</span>
            </Head>
            {asks.map((lv) => (
              <LevelRow key={`a-${lv.price}`} level={lv} side="ask" max={max} />
            ))}
            <SpreadRow>
              <Text variant="body" color="primary">
                {book.lastPrice || '—'}
              </Text>
              <Text variant="caption" color="secondary">
                {t('orderBook.mid')}
              </Text>
            </SpreadRow>
            {bids.map((lv) => (
              <LevelRow key={`b-${lv.price}`} level={lv} side="bid" max={max} />
            ))}
          </Ladder>
        </>
      )}
    </Panel>
  );
}
