import { useState } from 'react';
import { Alert, Modal } from 'antd';
import { useTranslation } from 'react-i18next';
import { SplitBar } from '@/components/atoms/SplitBar';
import { Text } from '@/components/atoms/Text';
import { DeskReadCard } from '@/components/molecules/DeskReadCard';
import { financeColors } from '@/styles/tokens';
import { upSharePct, windowMix } from './helpers';
import { DetailList, Head } from './BreadthOverview.styles';
import type { BreadthOverviewProps } from './BreadthOverview.types';

export function BreadthOverview({ data, error, isLoading = false }: BreadthOverviewProps) {
  const { t } = useTranslation('markets');
  const [open, setOpen] = useState(false);
  const hour = data?.windows?.find((w) => w.window === '1h') ?? data?.windows?.[0];
  const up = upSharePct(hour);

  return (
    <>
      <DeskReadCard
        title={t('breadth.title')}
        value={up == null ? '—' : `${Math.round(up)}%`}
        caption={
          hour?.alignment
            ? t(`breadth.align_${hour.alignment}`, { defaultValue: hour.alignment })
            : t('breadth.hour')
        }
        visual={
          <SplitBar
            ariaLabel={up == null ? t('breadth.title') : `${t('breadth.title')} ${Math.round(up)}`}
            parts={[
              { key: 'up', pct: Number(hour?.counts?.upPct) || 0, color: financeColors.up },
              { key: 'flat', pct: Number(hour?.counts?.flatPct) || 0, color: '#C7CDD8' },
              { key: 'down', pct: Number(hour?.counts?.downPct) || 0, color: financeColors.down },
            ]}
          />
        }
        isLoading={isLoading}
        onClick={() => setOpen(true)}
      />
      <Modal
        open={open}
        onCancel={() => setOpen(false)}
        footer={null}
        width={520}
        destroyOnHidden
        title={t('breadth.title')}
      >
        {error ? (
          <Alert type="info" showIcon message={t('breadth.title')} description={error} />
        ) : (
          <DetailList>
            <Text variant="caption" color="secondary">
              {t('breadth.subtitle', { n: data?.universe ?? '—' })}
            </Text>
            {(data?.windows ?? []).map((w) => (
              <Head key={w.window}>
                <Text variant="caption" color="secondary">
                  {w.window}
                  {w.alignment
                    ? ` · ${t(`breadth.align_${w.alignment}`, { defaultValue: w.alignment })}`
                    : ''}
                </Text>
                <Text variant="numeric" color="primary">
                  {windowMix(w)}
                </Text>
              </Head>
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
