import { Modal, Spin } from 'antd';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { heatFill } from '@/components/organisms/DerivativesHeatmap/helpers';
import type { MarketDerivativesGridCell } from '@/libs/api';
import { cvdScale, gridCoinPath, gridColumnLabel, gridDetailVenues, paintGridCell } from './helpers';
import {
  Cell,
  CellButton,
  CoinCell,
  CoinLink,
  DetailList,
  DetailRow,
  HeadCell,
  Scroll,
  Table,
} from './DerivativesTimeGrid.styles';
import type { DerivativesTimeGridProps } from './DerivativesTimeGrid.types';

type PickedCell = {
  symbol: string;
  time: string;
  filter: string;
};

export function DerivativesTimeGrid({ exchange, metric, range, grid, loading }: DerivativesTimeGridProps) {
  const { t, i18n } = useTranslation('derivatives');
  const rows = useMemo(() => grid?.rows ?? [], [grid?.rows]);
  const columns = grid?.columns ?? [];
  const full = useMemo(() => (metric === 'cvd' ? cvdScale(rows) : 1), [metric, rows]);
  const locale = i18n.language || 'en';
  const filter = `${exchange}|${metric}|${range}`;
  const [picked, setPicked] = useState<PickedCell | null>(null);
  const activePick = picked?.filter === filter ? picked : null;
  const selected = useMemo(() => {
    if (!activePick) return null;
    const row = rows.find((item) => item.symbol === activePick.symbol);
    const cell = row?.cells?.find((item) => item.time === activePick.time);
    if (!row || !cell) return null;
    return { row, cell, time: activePick.time };
  }, [activePick, rows]);

  if (!loading && rows.length === 0) {
    return <Text color="secondary">{t('derivatives:empty')}</Text>;
  }

  return (
    <Spin spinning={loading === true}>
      <Scroll>
        <Table aria-label={t('derivatives:gridLabel')}>
          <thead>
            <tr>
              <HeadCell scope="col">{t('derivatives:coin')}</HeadCell>
              {columns.map((column) => (
                <HeadCell key={column} scope="col">
                  {gridColumnLabel(column, range, locale)}
                </HeadCell>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => {
              const symbol = row.symbol ?? '';
              const coin = row.base || symbol;
              return (
                <tr key={symbol}>
                  <CoinCell>
                    <CoinLink to={gridCoinPath(exchange, symbol)}>{coin}</CoinLink>
                  </CoinCell>
                  {columns.map((column, index) => {
                    const cell = row.cells?.find((item) => item.time === column) ?? row.cells?.[index];
                    const painted = paintGridCell(
                      metric,
                      cell,
                      full,
                      t('derivatives:gridBlank'),
                      (venues) => t('derivatives:gridMissing', { venues }),
                    );
                    const timeLabel = gridColumnLabel(column, range, locale);
                    return (
                      <Cell
                        key={`${symbol}-${column}`}
                        $fill={heatFill(painted.tone, painted.strength)}
                        title={painted.title}
                        data-tone={painted.tone}
                      >
                        <CellButton
                          type="button"
                          aria-label={t('derivatives:gridCell', { coin, time: timeLabel, value: painted.text })}
                          onClick={() => setPicked({ symbol, time: column, filter })}
                        >
                          {painted.text}
                        </CellButton>
                      </Cell>
                    );
                  })}
                </tr>
              );
            })}
          </tbody>
        </Table>
      </Scroll>
      {selected ? (
        <CellDetail
          coin={selected.row.base || selected.row.symbol || ''}
          timeLabel={gridColumnLabel(selected.time, range, locale)}
          cell={selected.cell}
          onClose={() => setPicked(null)}
        />
      ) : null}
    </Spin>
  );
}

function CellDetail({
  coin,
  timeLabel,
  cell,
  onClose,
}: {
  coin: string;
  timeLabel: string;
  cell: MarketDerivativesGridCell;
  onClose: () => void;
}) {
  const { t } = useTranslation('derivatives');
  const venues = gridDetailVenues(cell);
  const showCombined = Boolean(cell.definite && cell.text);
  return (
    <Modal
      open
      onCancel={onClose}
      footer={null}
      width={480}
      destroyOnHidden
      title={t('derivatives:gridDetailTitle', { coin, time: timeLabel })}
    >
      <DetailList>
        <Text variant="caption" color="secondary">
          {t('derivatives:gridDetailHint')}
        </Text>
        {showCombined ? (
          <DetailRow>
            <Text color="secondary">{t('derivatives:gridCombined')}</Text>
            <Text variant="numeric">{cell.text}</Text>
          </DetailRow>
        ) : null}
        {venues.length === 0 && !showCombined ? (
          <Text color="secondary">{t('derivatives:gridNoVenues')}</Text>
        ) : (
          venues.map((venue) => (
            <DetailRow key={venue.exchange}>
              <Text>
                {t(`common:exchanges.${venue.exchange}`, { defaultValue: venue.exchange })}
              </Text>
              <Text variant="numeric" color={venue.definite ? 'primary' : 'secondary'}>
                {venue.definite ? venue.text : t('derivatives:gridVenueMissing')}
              </Text>
            </DetailRow>
          ))
        )}
      </DetailList>
    </Modal>
  );
}
