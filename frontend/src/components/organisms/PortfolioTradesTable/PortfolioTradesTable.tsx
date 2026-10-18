import { Button } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { situationChangeLines, situationLines } from '@/components/molecules/CoinSituation';
import { FillCostCheck } from '@/components/molecules/FillCostCheck';
import { FillMoves } from '@/components/molecules/FillMoves';
import type { PaperTrade } from '@/libs/api';
import { formatDateTime, formatPrice, formatSymbolDisplay } from '@/libs/utils';
import { DataTable, DataTableCard } from '@/styles/shared/dataTable.styles';
import type { PortfolioTradesTableProps } from './PortfolioTradesTable.types';

export function PortfolioTradesTable({ items, loading, onOpen }: PortfolioTradesTableProps) {
  const { t } = useTranslation(['portfolio', 'detail']);
  const columns: ColumnsType<PaperTrade> = [
    {
      title: t('trades.time'),
      dataIndex: 'createdAt',
      key: 't',
      render: (v: string | undefined) => (
        <Text variant="caption" mono>
          {formatDateTime(v)}
        </Text>
      ),
    },
    {
      title: t('positions.symbol'),
      key: 'sym',
      render: (_, row) => (
        <Button
          type="link"
          size="small"
          disabled={!onOpen || !row.exchange || !row.symbol}
          onClick={() => row.exchange && row.symbol && onOpen?.(row.exchange, row.symbol)}
        >
          {formatSymbolDisplay(row.symbol)}
        </Button>
      ),
    },
    {
      title: t('trades.side'),
      dataIndex: 'side',
      key: 'side',
      render: (v: string | undefined) =>
        v ? <BrandTag variant={v === 'buy' ? 'live' : 'status'}>{v}</BrandTag> : '—',
    },
    {
      title: t('positions.qty'),
      dataIndex: 'quantity',
      key: 'q',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
    {
      title: t('trades.price'),
      dataIndex: 'price',
      key: 'p',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
    {
      title: t('trades.fee'),
      dataIndex: 'fee',
      key: 'f',
      align: 'right',
      render: (v: number | undefined, row) => (
        <div>
          <Text variant="caption" mono>
            {formatPrice(v)}
          </Text>
          {row.costCheck ? <FillCostCheck check={row.costCheck} /> : null}
        </div>
      ),
    },
    {
      title: t('trades.situation'),
      key: 'situation',
      render: (_, row) => {
        const lines = situationLines(row.situation, t);
        const exits = row.side === 'sell' ? (row.left ?? []) : [];
        const exitLines = exits.flatMap((exit) =>
          situationChangeLines(exit.entry, row.situation, exit.comparison, 'exit', t),
        );
        if (lines.length === 0 && exitLines.length === 0) return '—';
        return (
          <div>
            {lines.length > 0 ? (
              <Text variant="caption" color="secondary">
                {t('detail:situation.atFill')}
              </Text>
            ) : null}
            {lines.map((line) => (
              <Text key={line} variant="caption">
                {line}
              </Text>
            ))}
            {exits.map((exit) => {
              const changed = situationChangeLines(exit.entry, row.situation, exit.comparison, 'exit', t);
              if (changed.length === 0) return null;
              return (
                <div key={exit.sourceTradeId ?? changed[0]}>
                  {exits.length > 1 ? (
                    <Text variant="caption" color="secondary">
                      {t('detail:situation.left.against', { time: formatDateTime(exit.entryAt) })}
                    </Text>
                  ) : null}
                  {changed.map((line) => (
                    <Text key={`${exit.sourceTradeId}-${line}`} variant="caption">
                      {line}
                    </Text>
                  ))}
                </div>
              );
            })}
          </div>
        );
      },
    },
    {
      title: t('moves.title'),
      key: 'moves',
      render: (_, row) => <FillMoves moves={row.moves} />,
    },
    {
      title: t('trades.realized'),
      dataIndex: 'realizedPnL',
      key: 'r',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
  ];

  if (!loading && items.length === 0) {
    return <DeskEmpty title={t('trades.empty')} />;
  }

  return (
    <DataTableCard>
      <DataTable
        rowKey={(r) => r.id ?? `${r.symbol}-${r.createdAt}`}
        size="small"
        pagination={{ pageSize: 10, hideOnSinglePage: true, simple: true }}
        loading={loading}
        columns={columns}
        dataSource={items}
        scroll={{ x: 720 }}
      />
    </DataTableCard>
  );
}
