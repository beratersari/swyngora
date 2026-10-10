import { Spin } from 'antd';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { heatFill } from '@/components/organisms/DerivativesHeatmap/helpers';
import { cvdScale, gridCoinPath, gridColumnLabel, paintGridCell } from './helpers';
import { Cell, CoinCell, CoinLink, HeadCell, Scroll, Table } from './DerivativesTimeGrid.styles';
import type { DerivativesTimeGridProps } from './DerivativesTimeGrid.types';

export function DerivativesTimeGrid({ exchange, metric, range, grid, loading }: DerivativesTimeGridProps) {
  const { t, i18n } = useTranslation('derivatives');
  const rows = grid?.rows ?? [];
  const columns = grid?.columns ?? [];
  const full = useMemo(() => (metric === 'cvd' ? cvdScale(rows) : 1), [metric, rows]);
  const locale = i18n.language || 'en';

  if (!loading && rows.length === 0) {
    return <Text color="secondary">{t('derivatives:empty')}</Text>;
  }

  return (
    <Spin spinning={loading}>
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
              return (
                <tr key={symbol}>
                  <CoinCell>
                    <CoinLink to={gridCoinPath(exchange, symbol)}>{row.base || symbol}</CoinLink>
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
                    return (
                      <Cell
                        key={`${symbol}-${column}`}
                        $fill={heatFill(painted.tone, painted.strength)}
                        title={painted.title}
                        data-tone={painted.tone}
                      >
                        {painted.text}
                      </Cell>
                    );
                  })}
                </tr>
              );
            })}
          </tbody>
        </Table>
      </Scroll>
    </Spin>
  );
}
