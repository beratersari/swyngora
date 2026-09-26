import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import type { VenueDivergenceBoardProps } from './VenueDivergenceBoard.types';

function names(list?: string[]) {
  return (list ?? []).filter(Boolean).join(', ');
}

export function VenueDivergenceBoard({ report }: VenueDivergenceBoardProps) {
  const { t } = useTranslation('detail');
  if (!report?.groups?.length && !report?.diffs?.length) return null;
  const outlier = (report.outliers ?? [])
    .map((ex) => {
      const lean = report.groups?.find((g) => g.exchanges?.includes(ex))?.lean;
      return lean ? `${ex} (${lean})` : ex;
    })
    .join(', ');
  const outlierSet = new Set(report.outliers ?? []);
  const together = (report.groups ?? []).filter((g) => (g.exchanges?.length ?? 0) > 1);
  const solo = (report.groups ?? []).filter(
    (g) => (g.exchanges?.length ?? 0) === 1 && !outlierSet.has(g.exchanges?.[0] ?? ''),
  );
  return (
    <>
      {together.map((g) => (
        <Text key={`${g.lean}-${names(g.exchanges)}`} variant="bodySm">
          {t('structure.sameWay')}: {names(g.exchanges)} ({g.lean})
        </Text>
      ))}
      {solo.map((g) => (
        <Text key={`${g.lean}-${names(g.exchanges)}`} variant="bodySm">
          {names(g.exchanges)} ({g.lean})
        </Text>
      ))}
      <Text variant="bodySm">
        {t('structure.differentWay')}: {outlier || t('structure.noOutlier')}
      </Text>
      {report.diffs?.map((d) => (
        <Text key={d.metric ?? d.label} variant="caption" color="secondary">
          {(d.label ?? d.metric) + ': ' + (d.readings ?? []).map((r) => `${r.exchange} ${r.value}`).join(' · ')}
        </Text>
      ))}
    </>
  );
}
