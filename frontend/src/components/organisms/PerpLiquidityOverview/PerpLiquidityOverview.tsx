import { useState } from 'react';
import { Alert, Modal } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { DeskGaugeCard } from '@/components/molecules/DeskGaugeCard';
import { formatCompactAmount } from '@/libs/utils';
import {
  bandLeaderLabel,
  bandShareLabel,
  bidSharePct,
  distanceBands,
  driverNames,
  leaderCoins,
  mixLine,
  shareLabel,
} from './helpers';
import { DetailList, DetailRow, Head } from './PerpLiquidityOverview.styles';
import type { PerpLiquidityOverviewProps } from './PerpLiquidityOverview.types';

export function PerpLiquidityOverview({ data, error, isLoading = false }: PerpLiquidityOverviewProps) {
  const { t } = useTranslation('markets');
  const [open, setOpen] = useState(false);
  const leaders = leaderCoins(data);
  const bands = distanceBands(data);
  const leanKey = data?.lean === 'bid' ? 'perpLiq.leanBid' : data?.lean === 'ask' ? 'perpLiq.leanAsk' : 'perpLiq.leanEven';
  const share = bidSharePct(data);
  const bid = formatCompactAmount(data?.bid);
  const ask = formatCompactAmount(data?.ask);

  return (
    <>
      <DeskGaugeCard
        title={t('perpLiq.title')}
        value={data?.lean ? t(leanKey) : '—'}
        caption={data?.bid || data?.ask ? `${bid} / ${ask}` : undefined}
        gaugeValue={share}
        minLabel={t('perpLiq.gaugeAsk')}
        maxLabel={t('perpLiq.gaugeBid')}
        isLoading={isLoading}
        onClick={() => setOpen(true)}
      />
      <Modal
        open={open}
        onCancel={() => setOpen(false)}
        footer={null}
        width={560}
        destroyOnHidden
        title={t('perpLiq.title')}
      >
        {error ? (
          <Alert type="info" showIcon message={t('perpLiq.title')} description={error} />
        ) : (
          <DetailList>
            <Text variant="caption" color="secondary">
              {t('perpLiq.subtitle', { pct: data?.rangePct ?? '1' })}
            </Text>
            <Head>
              <Text variant="caption" color="secondary">
                {t('perpLiq.bid')}
              </Text>
              <Text variant="numeric" color="primary">
                {bid}
              </Text>
            </Head>
            <Head>
              <Text variant="caption" color="secondary">
                {t('perpLiq.ask')}
              </Text>
              <Text variant="numeric" color="primary">
                {ask}
              </Text>
            </Head>
            <Head>
              <Text variant="caption" color="secondary">
                {t('perpLiq.lean')}
              </Text>
              <Text variant="numeric" color="primary">
                {data?.lean ? t(leanKey) : '—'}
              </Text>
            </Head>
            {leaders.length > 0 ? (
              <DetailList>
                {leaders.map((c) => (
                  <Head key={c.symbol}>
                    <Text variant="caption" color="secondary">
                      {c.base || c.symbol}
                    </Text>
                    <Text variant="numeric" color="primary">
                      {shareLabel(c.sharePct)}
                    </Text>
                  </Head>
                ))}
              </DetailList>
            ) : null}
            {bands.length > 0 ? (
              <>
                <Text variant="caption" color="secondary">
                  {t('perpLiq.bands')}
                </Text>
                {bands.map((b) => (
                  <DetailRow key={b.rangePct}>
                    <Head>
                      <Text variant="caption" color="secondary">
                        {t('perpLiq.band', { pct: b.rangePct })}
                      </Text>
                      <Text variant="numeric" color="primary">
                        {bandShareLabel(b, data?.rangePct)}
                      </Text>
                    </Head>
                    {bandLeaderLabel(b) ? (
                      <Text variant="caption" color="secondary">
                        {bandLeaderLabel(b)}
                      </Text>
                    ) : null}
                  </DetailRow>
                ))}
              </>
            ) : null}
            {(data?.windows ?? []).map((w) => (
              <DetailRow key={w.window}>
                <Head>
                  <Text variant="caption" color="secondary">
                    {t('perpLiq.window', { window: w.window })}
                  </Text>
                  <Text variant="numeric" color="primary">
                    {w.complete ? mixLine(w) : t('perpLiq.windowWait')}
                  </Text>
                </Head>
                {w.complete && driverNames(w) ? (
                  <Text variant="caption" color="secondary">
                    {t('perpLiq.drivers', { names: driverNames(w) })}
                  </Text>
                ) : null}
              </DetailRow>
            ))}
            {data?.summary ? (
              <Text variant="caption" color="secondary">
                {data.summary}
              </Text>
            ) : null}
          </DetailList>
        )}
      </Modal>
    </>
  );
}
