import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import {
  findPerpCoin,
  formatFundingPct,
  perpBidShareLine,
  perpDepthLine,
  formatSignedFundingPct,
  formatSignedTapeNum,
  formatTapeNum,
  shareMoveText,
  sharePctText,
  windowByName,
} from './TapePanel.helpers';
import { PositionSidePanel } from '@/components/organisms/PositionSidePanel';
import { Block, Panel, StatCard, StatsGrid, TitleRow } from './TapePanel.styles';
import type { TapePanelProps } from './TapePanel.types';

function formatOIWindow(
  w: { changeValue?: string; changeValuePct?: string; flow?: string } | undefined,
  t: (key: string, options?: Record<string, string>) => string,
): string {
  if (!w?.changeValue && !w?.changeValuePct) return '—';
  const amt = formatSignedTapeNum(w.changeValue);
  const pct = w.changeValuePct ? `(${w.changeValuePct}%)` : '';
  const flow = w.flow === 'opened' ? t('tape.oiOpened') : w.flow === 'closed' ? t('tape.oiClosed') : '';
  return [amt === '—' ? '' : amt, pct, flow].filter(Boolean).join(' ') || '—';
}

function formatAnchoredCvd(
  anchors: { anchor?: string; cvd?: string; complete?: boolean }[] | undefined,
  id: string,
  partial: string,
): string {
  const row = anchors?.find((a) => a.anchor === id);
  if (!row?.cvd) return '—';
  const value = formatTapeNum(row.cvd);
  return row.complete === false ? `${value} (${partial})` : value;
}

function Stat({
  label,
  value,
  isLoading,
}: {
  label: string;
  value: string;
  isLoading?: boolean;
}) {
  return (
    <StatCard>
      <Text variant="caption" color="secondary">
        {label}
      </Text>
      <Text variant="numeric" color="primary" isLoading={isLoading} skeletonWidth="70%">
        {value}
      </Text>
    </StatCard>
  );
}

export function TapePanel({
  openInterest,
  openInterestError,
  netPosition,
  netPositionError,
  liquidations,
  liquidationsError,
  cvd,
  cvdError,
  snapshot,
  snapshotError,
  participation,
  participationError,
  volumeShare,
  volumeShareError,
  perpLiquidity,
  perpLiquidityError,
  symbol,
  printSize,
  printSizeError,
  botTracker,
  botTrackerError,
  orderCount,
  orderCountError,
  pressure,
  pressureError,
  isLoading = false,
}: TapePanelProps) {
  const { t } = useTranslation('detail');
  const oi1h = windowByName(openInterest?.windows, '1h');
  const oi4h = windowByName(openInterest?.windows, '4h');
  const oi24h = windowByName(openInterest?.windows, '24h');
  const liq1h = windowByName(liquidations?.windows, '1h');
  const liq24h = windowByName(liquidations?.windows, '24h');
  const fundingPressure = openInterest?.funding?.pressure;
  const funding = fundingPressure ?? openInterest?.funding?.current;
  const funding1h = windowByName(openInterest?.funding?.pressureWindows, '1h');
  const funding24h = windowByName(openInterest?.funding?.pressureWindows, '24h');
  const tapePressure1h = windowByName(pressure?.windows, '1h');
  const nextSettle =
    fundingPressure?.nextFundingTime ??
    openInterest?.funding?.current?.nextFundingTime ??
    openInterest?.funding?.venues?.find((v) => v.current?.nextFundingTime)?.current
      ?.nextFundingTime;
  const partVenue = participation?.combined ?? participation?.venues?.[0];
  const part15 = windowByName(partVenue?.windows, '15m');
  const part1h = windowByName(partVenue?.windows, '1h');
  const part4h = windowByName(partVenue?.windows, '4h');
  const share1h = windowByName(volumeShare?.windows, '1h');
  const shareOf = (ex: string) => share1h?.venues?.find((v) => v.exchange === ex);
  const perpCoin = findPerpCoin(perpLiquidity?.coins, symbol);
  const perp1h = windowByName(perpLiquidity?.windows, '1h');
  const printVenue = printSize?.combined ?? printSize?.venues?.[0];
  const countVenue = orderCount?.combined ?? orderCount?.venues?.[0];
  const count1 = windowByName(countVenue?.windows, '1m');
  const count5 = windowByName(countVenue?.windows, '5m');
  const count15 = windowByName(countVenue?.windows, '15m');
  const print15 = windowByName(printVenue?.windows, '15m') ?? windowByName(printVenue?.windows, '5m');
  const cvdRow = cvd?.combined ?? cvd?.venues?.[0];
  const snap = snapshot?.combined ?? snapshot?.venues?.[0];
  const snap1h = snap?.windows?.find((w) => w.window === '1h');
  const lsVenue =
    openInterest?.longShort?.venues?.find((v) => v.exchange === 'binance') ??
    openInterest?.longShort?.venues?.[0];
  const lsSplit = lsVenue?.split;
  const lsGap = lsVenue?.gap ?? openInterest?.longShort?.gap;
  const lsGap1h = windowByName(lsVenue?.gapWindows ?? openInterest?.longShort?.gapWindows, '1h');
  const lsGap24h = windowByName(lsVenue?.gapWindows ?? openInterest?.longShort?.gapWindows, '24h');
  const lsDiv = lsVenue?.divergence ?? openInterest?.longShort?.divergence;

  return (
    <Panel>
      <TitleRow>
        <Text variant="h4" color="primary">
          {t('tape.title')}
        </Text>
        <Text variant="caption" color="secondary">
          {t('tape.subtitle')}
        </Text>
      </TitleRow>

      <Block>
        <Text variant="label" color="primary">
          {t('tape.oi')}
        </Text>
        {openInterestError ? (
          <Alert type="info" showIcon message={t('tape.oi')} description={openInterestError} />
        ) : (
          <StatsGrid>
            <Stat
              label={t('tape.oiNow')}
              value={formatTapeNum(openInterest?.current?.value ?? openInterest?.current?.contracts)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.oi1h')}
              value={formatOIWindow(oi1h, t)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.oi4h')}
              value={formatOIWindow(oi4h, t)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.oi24h')}
              value={formatOIWindow(oi24h, t)}
              isLoading={isLoading}
            />
          </StatsGrid>
        )}
        {openInterest?.venues?.map((v) => (
          <Text key={v.exchange} variant="caption" color="secondary">
            {v.exchange} {formatTapeNum(v.current?.value)} · 1h {formatOIWindow(windowByName(v.windows, '1h'), t)} · 4h{' '}
            {formatOIWindow(windowByName(v.windows, '4h'), t)} · 24h {formatOIWindow(windowByName(v.windows, '24h'), t)}
          </Text>
        ))}
        {oi1h && oi1h.complete === false ? (
          <Text variant="caption" color="secondary">
            {t('tape.incomplete')}
          </Text>
        ) : null}
        <PositionSidePanel data={netPosition} error={netPositionError} isLoading={isLoading} />
      </Block>

      <Block>
        <Text variant="label" color="primary">
          {t('tape.participation')}
        </Text>
        {participationError ? (
          <Alert type="info" showIcon message={t('tape.participation')} description={participationError} />
        ) : (
          <StatsGrid>
            <Stat
              label={t('tape.part15m')}
              value={part15?.ratioPct ? `${part15.ratioPct}%` : '—'}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.part1h')}
              value={part1h?.ratioPct ? `${part1h.ratioPct}%` : '—'}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.part4h')}
              value={part4h?.ratioPct ? `${part4h.ratioPct}%` : '—'}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.partVol15m')}
              value={formatTapeNum(part15?.volume)}
              isLoading={isLoading}
            />
          </StatsGrid>
        )}
        {part15?.oiDelta ? (
          <Text variant="caption" color="secondary">
            {t('tape.partOi15m', { value: formatSignedTapeNum(part15.oiDelta) })}
          </Text>
        ) : null}
        {partVenue?.summary || part15?.summary ? (
          <Text variant="caption" color="secondary">
            {partVenue?.summary || part15?.summary}
          </Text>
        ) : null}
      </Block>

      <Block>
        <Text variant="label" color="primary">
          {t('tape.volumeShare')}
        </Text>
        {volumeShareError ? (
          <Alert type="info" showIcon message={t('tape.volumeShare')} description={volumeShareError} />
        ) : (
          <StatsGrid>
            <Stat
              label={t('tape.shareBinance')}
              value={sharePctText(shareOf('binance'), t('tape.shareNoData'))}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.shareBybit')}
              value={sharePctText(shareOf('bybit'), t('tape.shareNoData'))}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.shareOkx')}
              value={sharePctText(shareOf('okx'), t('tape.shareNoData'))}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.share1h')}
              value={shareMoveText(share1h, {
                move: t('tape.shareMove', {
                  gained: (share1h?.gained ?? []).join(', ') || '—',
                  lost: (share1h?.lost ?? []).join(', ') || '—',
                }),
                missing: t('tape.shareMissing', { venues: (share1h?.missing ?? []).join(', ') }),
                stable: t('tape.shareStable'),
              })}
              isLoading={isLoading}
            />
          </StatsGrid>
        )}
        {share1h?.summary ? (
          <Text variant="caption" color="secondary">
            {share1h.summary}
          </Text>
        ) : null}
      </Block>

      <Block>
        <Text variant="label" color="primary">
          {t('tape.perpLiq')}
        </Text>
        {perpLiquidityError ? (
          <Alert type="info" showIcon message={t('tape.perpLiq')} description={perpLiquidityError} />
        ) : (
          <StatsGrid>
            <Stat
              label={t('tape.perpLiqBid')}
              value={formatTapeNum(perpLiquidity?.bid)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.perpLiqAsk')}
              value={formatTapeNum(perpLiquidity?.ask)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.perpLiqShare')}
              value={perpCoin?.sharePct ? `${perpCoin.sharePct}%` : '—'}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.perpLiqBidShare')}
              value={perpCoin?.bidSharePct ? `${perpCoin.bidSharePct}%` : '—'}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.perpLiq1h')}
              value={perpBidShareLine(perp1h)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.perpLiqDepth')}
              value={perpDepthLine(perp1h)}
              isLoading={isLoading}
            />
          </StatsGrid>
        )}
        {perpLiquidity?.summary ? (
          <Text variant="caption" color="secondary">
            {perpLiquidity.summary}
          </Text>
        ) : null}
      </Block>

      <Block>
        <Text variant="label" color="primary">
          {t('tape.funding')}
        </Text>
        {openInterestError && !funding ? (
          <Alert type="info" showIcon message={t('tape.funding')} description={openInterestError} />
        ) : (
          <>
            <StatsGrid>
              <Stat
                label={fundingPressure ? t('tape.pressure') : t('tape.nextRate')}
                value={formatFundingPct(funding?.ratePct)}
                isLoading={isLoading}
              />
              <Stat
                label={t('tape.payer')}
                value={funding?.payer ? t(`tape.payer_${funding.payer}`) : '—'}
                isLoading={isLoading}
              />
              <Stat
                label={t('tape.pressure1h')}
                value={formatSignedFundingPct(funding1h?.changePct)}
                isLoading={isLoading}
              />
              <Stat
                label={t('tape.pressure24h')}
                value={formatSignedFundingPct(funding24h?.changePct)}
                isLoading={isLoading}
              />
              <Stat
                label={t('tape.nextTime')}
                value={nextSettle ? new Date(nextSettle).toLocaleString() : '—'}
                isLoading={isLoading}
              />
            </StatsGrid>
            {fundingPressure?.summary ? (
              <Text variant="caption" color="secondary">
                {fundingPressure.summary}
              </Text>
            ) : null}
          </>
        )}
      </Block>

      <Block>
        <Text variant="label" color="primary">
          {t('tape.liquidations')}
        </Text>
        {liquidationsError ? (
          <Alert
            type="info"
            showIcon
            message={t('tape.liquidations')}
            description={liquidationsError}
          />
        ) : (
          <StatsGrid>
            <Stat
              label={t('tape.liq1h')}
              value={formatTapeNum(liq1h?.totalNotional)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.liq24h')}
              value={formatTapeNum(liq24h?.totalNotional)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.liqLong')}
              value={formatTapeNum(liq1h?.longNotional)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.liqShort')}
              value={formatTapeNum(liq1h?.shortNotional)}
              isLoading={isLoading}
            />
          </StatsGrid>
        )}
        {liq1h && liq1h.complete === false ? (
          <Text variant="caption" color="secondary">
            {t('tape.liqIncomplete')}
          </Text>
        ) : null}
      </Block>

      <Block>
        <Text variant="label" color="primary">
          {t('tape.printSize')}
        </Text>
        {printSizeError ? (
          <Alert type="info" showIcon message={t('tape.printSize')} description={printSizeError} />
        ) : (
          <StatsGrid>
            <Stat
              label={t('tape.avgBuy')}
              value={formatTapeNum(print15?.buy?.avg)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.avgSell')}
              value={formatTapeNum(print15?.sell?.avg)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.priorBuy')}
              value={formatTapeNum(print15?.buy?.priorAvg)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.priorSell')}
              value={formatTapeNum(print15?.sell?.priorAvg)}
              isLoading={isLoading}
            />
          </StatsGrid>
        )}
        {printVenue?.summary || print15?.summary ? (
          <Text variant="caption" color="secondary">
            {printVenue?.summary || print15?.summary}
          </Text>
        ) : null}
      </Block>

      <Block>
        <Text variant="label" color="primary">
          {t('tape.orderCount')}
        </Text>
        {orderCountError ? (
          <Alert type="info" showIcon message={t('tape.orderCount')} description={orderCountError} />
        ) : (
          <StatsGrid>
            <Stat label={t('tape.count5m')} value={count5?.ratio ?? '—'} isLoading={isLoading} />
            <Stat
              label={t('tape.countBuy')}
              value={count5?.buyCount === undefined ? '—' : String(count5.buyCount)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.countSell')}
              value={count5?.sellCount === undefined ? '—' : String(count5.sellCount)}
              isLoading={isLoading}
            />
            <Stat label={t('tape.count1m')} value={count1?.ratio ?? '—'} isLoading={isLoading} />
            <Stat label={t('tape.count15m')} value={count15?.ratio ?? '—'} isLoading={isLoading} />
          </StatsGrid>
        )}
        {countVenue?.summary || count5?.summary ? (
          <Text variant="caption" color="secondary">
            {countVenue?.summary || count5?.summary}
          </Text>
        ) : null}
      </Block>

      <Block>
        <Text variant="label" color="primary">
          {t('tape.botTracker')}
        </Text>
        {botTrackerError ? (
          <Alert type="info" showIcon message={t('tape.botTracker')} description={botTrackerError} />
        ) : (
          <StatsGrid>
            {(botTracker?.clusters ?? []).slice(0, 4).map((c) => (
              <Stat
                key={`${c.exchange}-${c.side}-${c.notional}`}
                label={c.side === 'sell' ? t('tape.botSellSize') : t('tape.botBuySize')}
                value={`${formatTapeNum(c.notional)} × ${c.count ?? 0}`}
                isLoading={isLoading}
              />
            ))}
            {(botTracker?.clusters ?? []).length === 0 ? (
              <Stat label={t('tape.botTracker')} value={t('tape.botNone')} isLoading={isLoading} />
            ) : (
              <>
                <Stat
                  label={t('tape.botBuyCount')}
                  value={String(botTracker?.buyCount ?? 0)}
                  isLoading={isLoading}
                />
                <Stat
                  label={t('tape.botSellCount')}
                  value={String(botTracker?.sellCount ?? 0)}
                  isLoading={isLoading}
                />
                <Stat
                  label={t('tape.botClipDelta')}
                  value={String(botTracker?.delta ?? 0)}
                  isLoading={isLoading}
                />
              </>
            )}
          </StatsGrid>
        )}
        {(botTracker?.clusters ?? []).length > 0 ? (
          <Text variant="caption" color="secondary">
            {t('tape.botCountsNote')}
          </Text>
        ) : null}
        {botTracker?.summary ? (
          <Text variant="caption" color="secondary">
            {botTracker.summary}
          </Text>
        ) : null}
      </Block>

      <Block>
        <Text variant="label" color="primary">
          {t('tape.cvd')}
        </Text>
        {cvdError ? (
          <Alert type="info" showIcon message={t('tape.cvd')} description={cvdError} />
        ) : (
          <StatsGrid>
            <Stat label={t('tape.cvdLast')} value={formatTapeNum(cvdRow?.lastCvd)} isLoading={isLoading} />
            <Stat
              label={t('tape.cvdDay')}
              value={formatAnchoredCvd(cvdRow?.anchors, 'day', t('tape.cvdPartial'))}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.cvd4h')}
              value={formatAnchoredCvd(cvdRow?.anchors, '4h', t('tape.cvdPartial'))}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.cvdSummary')}
              value={cvdRow?.summary || cvd?.summary || '—'}
              isLoading={isLoading}
            />
          </StatsGrid>
        )}
        {cvdRow?.anchors?.map((a) =>
          a.summary ? (
            <Text key={a.anchor} variant="caption" color="secondary">
              {a.summary}
            </Text>
          ) : null,
        )}
        {cvdRow?.error ? (
          <Text variant="caption" color="secondary">
            {cvdRow.error}
          </Text>
        ) : null}
      </Block>

      <Block>
        <Text variant="label" color="primary">
          {t('tape.netPressure')}
        </Text>
        {pressureError ? (
          <Alert type="info" showIcon message={t('tape.netPressure')} description={pressureError} />
        ) : (
          <StatsGrid>
            <Stat
              label={t('tape.netNow')}
              value={pressure?.current?.value ?? '—'}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.netLean')}
              value={
                pressure?.current?.lean
                  ? t(`tape.net_${pressure.current.lean}`, { defaultValue: pressure.current.lean })
                  : '—'
              }
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.net1h')}
              value={pressure?.change1h ?? tapePressure1h?.change ?? '—'}
              isLoading={isLoading}
            />
          </StatsGrid>
        )}
        {pressure?.current?.headline ? (
          <Text variant="caption" color="secondary">
            {pressure.current.headline}
          </Text>
        ) : null}
      </Block>

      <Block>
        <Text variant="label" color="primary">
          {t('tape.positioning')}
        </Text>
        {snapshotError ? (
          <Alert type="info" showIcon message={t('tape.positioning')} description={snapshotError} />
        ) : (
          <StatsGrid>
            <Stat
              label={t('tape.longPct')}
              value={snap?.longPct ? `${snap.longPct}%` : '—'}
              isLoading={isLoading}
            />
            {lsVenue?.topPositions ? (
              <Stat
                label={t('tape.topPosLongPct')}
                value={`${lsVenue.topPositions.longPct}%`}
                isLoading={isLoading}
              />
            ) : null}
            {lsGap ? (
              <Stat
                label={t('tape.crowdGap')}
                value={`${lsGap.gapPct} pp`}
                isLoading={isLoading}
              />
            ) : null}
            {lsGap1h?.kind ? (
              <Stat
                label={t('tape.gap1h')}
                value={t(`tape.gap_${lsGap1h.kind}`, { defaultValue: lsGap1h.kind })}
                isLoading={isLoading}
              />
            ) : null}
            {lsGap24h?.kind ? (
              <Stat
                label={t('tape.gap24h')}
                value={t(`tape.gap_${lsGap24h.kind}`, { defaultValue: lsGap24h.kind })}
                isLoading={isLoading}
              />
            ) : null}
            {lsDiv ? (
              <Stat
                label={t('tape.lsDiv')}
                value={lsDiv.duration ? `${lsDiv.duration}` : '—'}
                isLoading={isLoading}
              />
            ) : null}
            {lsDiv?.since ? (
              <Stat
                label={t('tape.lsDivSince')}
                value={new Date(lsDiv.since).toLocaleTimeString()}
                isLoading={isLoading}
              />
            ) : null}
            <Stat
              label={t('tape.taker1h')}
              value={snap1h?.taker?.dominant ?? '—'}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.takerShare')}
              value={snap1h?.taker?.buyShare ?? '—'}
              isLoading={isLoading}
            />
          </StatsGrid>
        )}
        {lsGap?.headline ? (
          <Text variant="caption" color="secondary">
            {lsGap.headline}
            {lsGap1h?.kind && lsGap1h.kind !== 'stable'
              ? ` ${t(`tape.gap_${lsGap1h.kind}`, { defaultValue: lsGap1h.kind })} 1h.`
              : ''}
          </Text>
        ) : null}
        {lsDiv?.headline ? (
          <Text variant="caption" color="secondary">
            {lsDiv.headline}
            {lsDiv.detail ? ` ${lsDiv.detail}` : ''}
          </Text>
        ) : null}
        {lsSplit?.alignment === 'opposite' && lsSplit.headline ? (
          <Text variant="caption" color="secondary">
            {lsSplit.headline}
            {lsSplit.detail ? ` ${lsSplit.detail}` : ''}
          </Text>
        ) : null}
        {snapshot?.summary ? (
          <Text variant="caption" color="secondary">
            {snapshot.summary}
          </Text>
        ) : null}
      </Block>
    </Panel>
  );
}
