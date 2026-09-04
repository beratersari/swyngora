import { Skeleton, Table } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import type { ForensicsScanHit } from '@/libs/types/forensics.types';
import { Stack, TableWrap } from './ForensicsNowScan.styles';
import type { ForensicsNowScanProps } from './ForensicsNowScan.types';

function horizonCell(hit: ForensicsScanHit, id: string): string {
  const h = hit.afterHorizons?.find((row) => row.horizon === id);
  if (!h) return '—';
  const bits = [`↑${h.up ?? 0}`, `↓${h.down ?? 0}`];
  if (h.medianPct) bits.push(h.medianPct);
  return bits.join(' ');
}

export function ForensicsNowScan({ scan, isLoading, onOpenSymbol }: ForensicsNowScanProps) {
  const { t } = useTranslation('forensics');
  if (isLoading) {
    return <Skeleton active paragraph={{ rows: 6 }} />;
  }
  const hits = scan?.hits ?? [];
  return (
    <Stack>
      {scan?.summary ? <Text variant="bodySm">{scan.summary}</Text> : null}
      {scan?.note ? (
        <Text variant="caption" color="secondary">
          {scan.note}
        </Text>
      ) : null}
      {hits.length === 0 ? (
        <Text variant="caption" color="secondary">
          {t('scan.empty')}
        </Text>
      ) : (
        <TableWrap>
          <Table<ForensicsScanHit>
            size="small"
            pagination={false}
            rowKey={(row) => row.symbol ?? ''}
            dataSource={hits}
            onRow={(row) => ({
              onClick: () => {
                if (row.symbol) onOpenSymbol?.(row.symbol);
              },
              style: onOpenSymbol ? { cursor: 'pointer' } : undefined,
            })}
            columns={[
              { title: t('symbol'), dataIndex: 'symbol', key: 'symbol' },
              { title: t('similarity'), dataIndex: 'similarity', key: 'similarity' },
              {
                title: t('scan.lean'),
                dataIndex: 'lean',
                key: 'lean',
                render: (lean: string) => t(`scan.leans.${lean}`, { defaultValue: lean }),
              },
              {
                title: t('scan.after1h'),
                key: 'h1',
                render: (_: unknown, row) => horizonCell(row, '1h'),
              },
              {
                title: t('scan.after4h'),
                key: 'h4',
                render: (_: unknown, row) => horizonCell(row, '4h'),
              },
              { title: t('scan.afterStory'), dataIndex: 'afterStory', key: 'afterStory' },
            ]}
          />
        </TableWrap>
      )}
    </Stack>
  );
}
