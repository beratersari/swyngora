import { useState } from 'react';
import { Button, InputNumber } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import type { PortfolioStressReport } from '@/libs/api';

type Move = { asset: string; movePct: number };

type PortfolioStressPanelProps = {
  disabled?: boolean;
  isLoading?: boolean;
  report?: PortfolioStressReport | null;
  errorMessage?: string | null;
  onPreview: (moves: Move[]) => void;
};

const startMoves: Move[] = [
  { asset: 'BTC', movePct: -10 },
  { asset: 'ETH', movePct: -15 },
];

function money(n: number | undefined, signed = false): string {
  if (n == null || !Number.isFinite(n)) return '—';
  const body = Math.abs(n).toLocaleString(undefined, { maximumFractionDigits: 2 });
  if (!signed) return n < 0 ? `-${body}` : body;
  if (n > 0) return `+${body}`;
  if (n < 0) return `-${body}`;
  return body;
}

export function PortfolioStressPanel({ disabled, isLoading, report, errorMessage, onPreview }: PortfolioStressPanelProps) {
  const { t } = useTranslation('portfolio');
  const [moves, setMoves] = useState<Move[]>(startMoves);

  return (
    <section data-testid="portfolio-stress">
      <Text variant="h4" color="primary">
        {t('stress.title')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('stress.hint')}
      </Text>
      {moves.map((row, i) => (
        <div key={row.asset} style={{ display: 'flex', gap: 8, marginTop: 8 }}>
          <Text variant="body" color="primary">
            {row.asset}
          </Text>
          <InputNumber
            aria-label={t('stress.move', { asset: row.asset })}
            value={row.movePct}
            step={1}
            onChange={(v) => {
              const next = [...moves];
              next[i] = { ...row, movePct: typeof v === 'number' ? v : row.movePct };
              setMoves(next);
            }}
          />
        </div>
      ))}
      <Button
        style={{ marginTop: 8 }}
        type="primary"
        disabled={disabled}
        loading={isLoading}
        onClick={() => onPreview(moves)}
      >
        {t('stress.preview')}
      </Button>
      {errorMessage ? (
        <Text variant="caption" color="secondary">
          {errorMessage}
        </Text>
      ) : null}
      {report ? (
        <div data-testid="portfolio-stress-result">
          <Text variant="body" color="primary">
            {t('stress.equity', { now: money(report.equity), next: money(report.shockedEquity), delta: money(report.equityDelta, true) })}
          </Text>
          <Text variant="body" color="primary">
            {t('stress.pnl', { now: money(report.totalPnl, true), next: money(report.shockedTotalPnl, true) })}
          </Text>
          <Text variant="body" color="primary">
            {t('stress.margin', { now: money(report.marginEquity), next: money(report.shockedMarginEquity) })}
          </Text>
          {(report.liquidated ?? 0) > 0 ? (
            <Text variant="caption" color="secondary">
              {t('stress.liquidated', { n: report.liquidated })}
            </Text>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}
