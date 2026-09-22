import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { sideLine } from './helpers';
import { Block, Row } from './PositionSidePanel.styles';
import type { PositionSidePanelProps } from './PositionSidePanel.types';

const WINDOWS = ['15m', '1h', '4h'] as const;

export function PositionSidePanel({ data, error, isLoading = false }: PositionSidePanelProps) {
  const { t } = useTranslation('detail');
  return (
    <Block>
      <Text variant="label" color="primary">
        {t('tape.netPosition')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('tape.netPositionHint')}
      </Text>
      {error ? (
        <Alert type="info" showIcon message={t('tape.netPosition')} description={error} />
      ) : (
        WINDOWS.map((id) => {
          const row = data?.windows?.find((w) => w.window === id);
          return (
            <Row key={id}>
              <Text variant="caption" color="secondary">
                {id}
              </Text>
              <Text variant="numeric" color="primary" isLoading={isLoading} skeletonWidth="70%">
                {row ? sideLine(row.netLong, row.netShort) : '—'}
              </Text>
            </Row>
          );
        })
      )}
    </Block>
  );
}
