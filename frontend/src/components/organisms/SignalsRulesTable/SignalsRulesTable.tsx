import { Button, Popconfirm } from 'antd';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import type { ScannerRule } from '@/libs/api';
import { describeRule, ruleTypeShort } from '@/libs/utils';
import { DataTable, DataTableCard } from '@/styles/shared/dataTable.styles';
import type { SignalsRulesTableProps } from './SignalsRulesTable.types';

export function SignalsRulesTable({
  items,
  loading,
  deleteLoading,
  onDelete,
}: SignalsRulesTableProps) {
  const { t } = useTranslation(['signals', 'common']);

  const columns: ColumnsType<ScannerRule> = [
    {
      title: t('signals:rules.type'),
      key: 'type',
      render: (_, row) => <BrandTag variant="status">{ruleTypeShort(row.type)}</BrandTag>,
    },
    {
      title: t('signals:interval'),
      dataIndex: 'interval',
      key: 'interval',
      render: (iv: string) => (
        <Text variant="caption" mono>
          {iv}
        </Text>
      ),
    },
    {
      title: t('signals:rules.params'),
      key: 'params',
      render: (_, row) => <Text variant="body">{describeRule(row)}</Text>,
    },
    {
      title: t('signals:rules.enabled'),
      key: 'enabled',
      render: (_, row) => (
        <BrandTag variant={row.enabled ? 'live' : 'paused'}>
          {row.enabled ? t('signals:rules.on') : t('signals:rules.off')}
        </BrandTag>
      ),
    },
    {
      title: t('signals:actions'),
      key: 'actions',
      render: (_, row) => (
        <Popconfirm
          title={t('signals:rules.deleteConfirm')}
          okText={t('signals:rules.delete')}
          cancelText={t('common:actions.cancel')}
          onConfirm={() => row.id && onDelete(row.id)}
        >
          <Button size="small" danger loading={deleteLoading}>
            {t('signals:rules.delete')}
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <DataTableCard>
      <DataTable
        rowKey={(r) => r.id}
        loading={loading}
        dataSource={items}
        columns={columns}
        pagination={false}
        locale={{ emptyText: <DeskEmpty title={t('signals:rules.empty')} /> }}
        size="small"
        scroll={{ x: 640 }}
      />
    </DataTableCard>
  );
}
