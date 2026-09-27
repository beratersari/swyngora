import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatRecovery } from './helpers';
import { List } from './SweepRecoveryList.styles';
import type { SweepRecoveryListProps } from './SweepRecoveryList.types';

export function SweepRecoveryList({ report }: SweepRecoveryListProps) {
  const { t } = useTranslation('detail');
  const rows = report?.rows ?? [];
  if (rows.length === 0) return null;
  return (
    <List data-testid="sweep-recovery">
      {rows.map((row) => {
        const side = row.side === 'low' ? t('flow.recovery.low') : t('flow.recovery.high');
        const label = row.recovered
          ? t('flow.recovery.row', { side, level: row.level ?? '—', time: formatRecovery(row.recoverySeconds) })
          : t('flow.recovery.pending', { side, level: row.level ?? '—' });
        return (
          <li key={`${row.exchange}-${row.side}-${row.level}-${row.recoverySeconds}`}>
            <Text variant="caption" color="primary">
              {label}
            </Text>
          </li>
        );
      })}
    </List>
  );
}
