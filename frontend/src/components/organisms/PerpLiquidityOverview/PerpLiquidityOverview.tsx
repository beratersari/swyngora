import { useState } from 'react';
import { Alert, Modal } from 'antd';
import { useTranslation } from 'react-i18next';
import { SplitBar } from '@/components/atoms/SplitBar';
import { Text } from '@/components/atoms/Text';
import { DeskReadCard } from '@/components/molecules/DeskReadCard';
import { VenueCoverage } from '@/components/molecules/VenueCoverage';
import { financeColors } from '@/styles/tokens';
import { PerpLiquidityDepthChart } from './PerpLiquidityDepthChart';
import { DetailList, DetailRow, Head, Legend, Swatch } from './PerpLiquidityOverview.styles';
import { formatCompactAmount } from '@/libs/utils';
import {
  bandLeaderLabel,
  bandShareLabel,
  bidSharePct,
  distanceBands,
  depthLine,
  moveAfterLine,
  moveBookLine,
  moveDriverLine,
  moveDuringLine,
  moveVenueLine,
  moveFollowTotal,
  driverNames,
  sizeDriverLine,
  leaderCoins,
  mixLine,
  shareLabel,
} from './helpers';
import type { PerpLiquidityOverviewProps } from './PerpLiquidityOverview.types';

export function PerpLiquidityOverview({ data, error, isLoading = false }: PerpLiquidityOverviewProps) {
  const { t } = useTranslation(['markets', 'common']);
  const [open, setOpen] = useState(false);
  const leaders = leaderCoins(data);
  const bands = distanceBands(data);
  const leanKey = data?.lean === 'bid' ? 'perpLiq.leanBid' : data?.lean === 'ask' ? 'perpLiq.leanAsk' : 'perpLiq.leanEven';
  const share = bidSharePct(data);
  const bid = formatCompactAmount(data?.bid);
  const ask = formatCompactAmount(data?.ask);

  return (
    <>
      <DeskReadCard
        title={t('perpLiq.title')}
        value={data?.definite === false || !data?.lean ? '—' : t(leanKey)}
        caption={
          data?.definite === false
            ? t('common:coverage.incomplete')
            : data?.bid || data?.ask
              ? `${bid} / ${ask}`
              : undefined
        }
        visual={
          data?.definite === false ? undefined : (
          <SplitBar
            ariaLabel={share == null ? t('perpLiq.title') : `${t('perpLiq.title')} ${Math.round(share)}`}
            parts={[
              { key: 'bid', pct: Number(data?.bid) || 0, color: financeColors.up },
              { key: 'ask', pct: Number(data?.ask) || 0, color: financeColors.down },
            ]}
          />
          )
        }
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
            {data?.definite === false ? <VenueCoverage coverage={data.coverage} /> : null}
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
            <Text variant="caption" color="secondary">
              {t('perpLiq.history')}
            </Text>
            {(data?.history?.length ?? 0) >= 2 ? (
              <Legend>
                <Text variant="caption" color="secondary">
                  <Swatch $color={financeColors.cobalt} />
                  {t('perpLiq.book')}
                </Text>
                <Text variant="caption" color="secondary">
                  <Swatch $color={financeColors.warning} />
                  {t('perpLiq.btc')}
                </Text>
              </Legend>
            ) : null}
            {(data?.history?.length ?? 0) >= 2 ? (
              <PerpLiquidityDepthChart history={data?.history} />
            ) : (
              <Text variant="caption" color="secondary">
                {t('perpLiq.historyWait')}
              </Text>
            )}
            {(data?.history?.length ?? 0) >= 2 ? (
              <>
                <Text variant="caption" color="secondary">
                  {t('perpLiq.moves')}
                </Text>
                {(data?.moves?.length ?? 0) === 0 ? (
                  <Text variant="caption" color="secondary">
                    {t('perpLiq.movesWait')}
                  </Text>
                ) : (
                  (data?.moves ?? []).map((m) => (
                    <DetailRow key={`${m.kind}-${m.from}-${m.to}`}>
                      <Head>
                        <Text variant="caption" color="secondary">
                          {t(m.kind === 'drop' ? 'perpLiq.moveDrop' : 'perpLiq.moveJump')}
                        </Text>
                        <Text variant="numeric" color="primary">
                          {moveBookLine(m)}
                        </Text>
                      </Head>
                      {moveDuringLine(m) ? (
                        <Text variant="caption" color="secondary">
                          {t('perpLiq.moveBtc', { line: moveDuringLine(m) })}
                        </Text>
                      ) : null}
                      {moveAfterLine(m) ? (
                        <Text variant="caption" color="secondary">
                          {t('perpLiq.moveAfter', { line: moveAfterLine(m) })}
                        </Text>
                      ) : null}
                      {m.follow && moveFollowTotal(m) ? (
                        <Text variant="caption" color="secondary">
                          {t(`perpLiq.moveFollow_${m.follow}`, { total: moveFollowTotal(m) })}
                        </Text>
                      ) : null}
                      {moveDriverLine(m) ? (
                        <Text variant="caption" color="secondary">
                          {t('perpLiq.moveCoins', { names: moveDriverLine(m) })}
                          {m.span ? ` · ${t(`perpLiq.moveSpan_${m.span}`)}` : ''}
                        </Text>
                      ) : null}
                      {moveVenueLine(m) ? (
                        <Text variant="caption" color="secondary">
                          {t('perpLiq.moveVenues', { names: moveVenueLine(m) })}
                          {m.venueSpan ? ` · ${t(`perpLiq.moveVenue_${m.venueSpan}`)}` : ''}
                        </Text>
                      ) : null}
                    </DetailRow>
                  ))
                )}
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
                {w.complete && depthLine(w) ? (
                  <Head>
                    <Text variant="caption" color="secondary">
                      {t('perpLiq.depth', { window: w.window })}
                    </Text>
                    <Text variant="numeric" color="primary">
                      {depthLine(w)}
                      {w.depthKind && w.depthKind !== 'steady' ? ` · ${t(`perpLiq.${w.depthKind}`)}` : ''}
                    </Text>
                  </Head>
                ) : null}
                {w.complete && sizeDriverLine(w) ? (
                  <Text variant="caption" color="secondary">
                    {t('perpLiq.sizeDrivers', { names: sizeDriverLine(w) })}
                  </Text>
                ) : null}
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
