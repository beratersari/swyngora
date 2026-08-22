import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatFundingPct, formatTapeNum, windowByName } from './TapePanel.helpers';
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
  isLoading = false,
}: TapePanelProps) {
  const { t } = useTranslation('detail');
  const oi1h = windowByName(openInterest?.windows, '1h');
  const oi24h = windowByName(openInterest?.windows, '24h');
  const liq1h = windowByName(liquidations?.windows, '1h');
  const liq24h = windowByName(liquidations?.windows, '24h');
  const funding = openInterest?.funding?.current;
  const cvdRow = cvd?.combined ?? cvd?.venues?.[0];

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
          <StatsGrid>
            <Stat
              label={t('tape.nextRate')}
              value={formatFundingPct(funding?.ratePct)}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.payer')}
              value={funding?.payer ? t(`tape.payer_${funding.payer}`) : '—'}
              isLoading={isLoading}
            />
            <Stat
              label={t('tape.nextTime')}
              value={
                funding?.nextFundingTime
                  ? new Date(funding.nextFundingTime).toLocaleString()
                  : '—'
              }
              isLoading={isLoading}
            />
          </StatsGrid>
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
    </Panel>
  );
}
