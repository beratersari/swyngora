import { useState } from 'react';
import { InfoCircleOutlined } from '@ant-design/icons';
import { Alert, Modal, Tabs } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { FEAR_GREED_TAB_DRIVERS, FEAR_GREED_TAB_HISTORY } from './constants';
import { fearGreedTone, fearGreedWindowLine, recentFearGreed } from './helpers';
import {
  Chip,
  ChipText,
  FactorHead,
  FactorList,
  FactorRow,
  HistoryBar,
  HistoryRow,
  InfoButton,
  ModalTitle,
} from './FearGreedPanel.styles';
import type { FearGreedPanelProps } from './FearGreedPanel.types';

function barTone(level?: string): 'up' | 'down' | 'flat' {
  const tone = fearGreedTone(level);
  if (tone === 'success') return 'up';
  if (tone === 'error') return 'down';
  return 'flat';
}

function factorTone(toward?: string) {
  if (toward === 'greed') return fearGreedTone('greed');
  if (toward === 'fear') return fearGreedTone('fear');
  return fearGreedTone('neutral');
}

export function FearGreedPanel({ data, error, isLoading = false }: FearGreedPanelProps) {
  const { t } = useTranslation('markets');
  const [open, setOpen] = useState(false);
  const [about, setAbout] = useState(false);
  const tone = fearGreedTone(data?.level);
  const levelLabel = data?.level
    ? t(`fearGreed.level_${data.level}`, { defaultValue: data.label ?? data.level })
    : '—';
  const bars = recentFearGreed(data?.history);
  const windows = data?.windows ?? [];

  const close = () => {
    setOpen(false);
    setAbout(false);
  };

  return (
    <>
      <Chip type="button" onClick={() => setOpen(true)} aria-haspopup="dialog">
        <Text variant="label" color="primary">
          {t('fearGreed.title')}
        </Text>
        <ChipText>
          <Text variant="h3" color={tone} isLoading={isLoading} skeletonWidth={36}>
            {error ? '—' : (data?.score ?? '—')}
          </Text>
          <Text variant="numeric" color={tone} isLoading={isLoading} skeletonWidth={72}>
            {error ? '' : levelLabel}
          </Text>
        </ChipText>
      </Chip>
      <Modal
        open={open}
        onCancel={close}
        footer={null}
        width={560}
        destroyOnHidden
        title={
          <ModalTitle>
            <Text variant="label" color="primary">
              {t('fearGreed.title')}
            </Text>
            <Text variant="h4" color={tone}>
              {data?.score ?? '—'}
            </Text>
            <Text variant="numeric" color={tone}>
              {levelLabel}
            </Text>
            <InfoButton
              type="button"
              aria-label={t('fearGreed.info')}
              aria-expanded={about}
              onClick={() => setAbout((on) => !on)}
            >
              <InfoCircleOutlined />
            </InfoButton>
          </ModalTitle>
        }
      >
        {about ? (
          <Text variant="caption" color="secondary">
            {t('fearGreed.about')}
          </Text>
        ) : null}
        {error ? (
          <Alert type="info" showIcon message={t('fearGreed.title')} description={error} />
        ) : (
          <Tabs
            items={[
              {
                key: FEAR_GREED_TAB_DRIVERS,
                label: t('fearGreed.tabDrivers'),
                children: (
                  <FactorList>
                    {data?.summary ? (
                      <Text variant="caption" color="secondary">
                        {data.summary}
                      </Text>
                    ) : null}
                    <Text variant="caption" color="secondary">
                      {t('fearGreed.inputs', { used: data?.used ?? '—', total: data?.total ?? 8 })}
                    </Text>
                    {(data?.factors?.length ?? 0) === 0 ? (
                      <Text variant="caption" color="secondary">
                        {t('fearGreed.even')}
                      </Text>
                    ) : (
                      data?.factors?.map((factor) => (
                        <FactorRow key={factor.id ?? factor.name}>
                          <FactorHead>
                            <Text variant="label" color="primary">
                              {t(`fearGreed.factor_${factor.id ?? ''}`, {
                                defaultValue: factor.name ?? factor.id,
                              })}
                            </Text>
                            <Text variant="numeric" color={factorTone(factor.toward)}>
                              {factor.pull ?? '—'}
                            </Text>
                          </FactorHead>
                          {factor.detail ? (
                            <Text variant="caption" color="secondary">
                              {factor.detail}
                            </Text>
                          ) : null}
                        </FactorRow>
                      ))
                    )}
                  </FactorList>
                ),
              },
              {
                key: FEAR_GREED_TAB_HISTORY,
                label: t('fearGreed.tabHistory'),
                children: (
                  <FactorList>
                    {windows.map((w) => {
                      const line = fearGreedWindowLine(w);
                      return (
                        <FactorHead key={w.window}>
                          <Text variant="caption" color="secondary">
                            {w.window}
                          </Text>
                          <Text variant="numeric" color="primary">
                            {line ?? t('fearGreed.windowWait')}
                          </Text>
                        </FactorHead>
                      );
                    })}
                    {bars.length > 0 ? (
                      <HistoryRow aria-hidden>
                        {bars.map((point, i) => (
                          <HistoryBar
                            key={`${point.time ?? i}`}
                            $height={Math.round(((point.score ?? 0) / 100) * 36)}
                            $tone={barTone(point.level)}
                            title={`${point.score ?? ''}`}
                          />
                        ))}
                      </HistoryRow>
                    ) : (
                      <Text variant="caption" color="secondary">
                        {t('fearGreed.historyEmpty')}
                      </Text>
                    )}
                  </FactorList>
                ),
              },
            ]}
          />
        )}
      </Modal>
    </>
  );
}
