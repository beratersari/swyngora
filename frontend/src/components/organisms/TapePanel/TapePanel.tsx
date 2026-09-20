import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatFundingPct, formatSignedFundingPct, formatTapeNum, windowByName } from './TapePanel.helpers';
import { Block, Panel, StatCard, StatsGrid, TitleRow } from './TapePanel.styles';
import type { TapePanelProps } from './TapePanel.types';

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
  liquidations,
  liquidationsError,
  cvd,
  cvdError,
  snapshot,
  snapshotError,
  printSize,
  printSizeError,
  isLoading = false,
}: TapePanelProps) {
  const { t } = useTranslation('detail');
  const oi1h = windowByName(openInterest?.windows, '1h');
  const oi24h = windowByName(openInterest?.windows, '24h');
  const liq1h = windowByName(liquidations?.windows, '1h');
  const liq24h = windowByName(liquidations?.windows, '24h');
  const pressure = openInterest?.funding?.pressure;
  const funding = pressure ?? openInterest?.funding?.current;
  const pressure1h = windowByName(openInterest?.funding?.pressureWindows, '1h');
  const pressure24h = windowByName(openInterest?.funding?.pressureWindows, '24h');
  const nextSettle =
    pressure?.nextFundingTime ??
    openInterest?.funding?.current?.nextFundingTime ??
    openInterest?.funding?.venues?.find((v) => v.current?.nextFundingTime)?.current
      ?.nextFundingTime;
  const printVenue = printSize?.combined ?? printSize?.venues?.[0];
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
              value={formatTapeNum(oi1h?.changeValuePct ?? oi1h?.changePct)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.oi24h')}
              value={formatTapeNum(oi24h?.changeValuePct ?? oi24h?.changePct)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.venues')}
              value={openInterest?.venueCount != null ? String(openInterest.venueCount) : '—'}
              isLoading={isLoading}
            />
          </StatsGrid>
        )}
        {oi1h && oi1h.complete === false ? (
          <Text variant="caption" color="secondary">
            {t('tape.incomplete')}
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
                label={pressure ? t('tape.pressure') : t('tape.nextRate')}
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
                value={formatSignedFundingPct(pressure1h?.changePct)}
                isLoading={isLoading}
              />
              <Stat
                label={t('tape.pressure24h')}
                value={formatSignedFundingPct(pressure24h?.changePct)}
                isLoading={isLoading}
              />
              <Stat
                label={t('tape.nextTime')}
                value={nextSettle ? new Date(nextSettle).toLocaleString() : '—'}
                isLoading={isLoading}
              />
            </StatsGrid>
            {pressure?.summary ? (
              <Text variant="caption" color="secondary">
                {pressure.summary}
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
          {t('tape.cvd')}
        </Text>
        {cvdError ? (
          <Alert type="info" showIcon message={t('tape.cvd')} description={cvdError} />
        ) : (
          <StatsGrid>
            <Stat label={t('tape.cvdLast')} value={formatTapeNum(cvdRow?.lastCvd)} isLoading={isLoading} />
            <Stat
              label={t('tape.cvdSummary')}
              value={cvdRow?.summary || cvd?.summary || '—'}
              isLoading={isLoading}
            />
          </StatsGrid>
        )}
        {cvdRow?.error ? (
          <Text variant="caption" color="secondary">
            {cvdRow.error}
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
