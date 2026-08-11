import styled from 'styled-components';

export const Panel = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  padding: ${({ theme }) => theme.spacing[4]}px;
  background: ${({ theme }) => theme.semantic.bg.muted};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.lg}px;
  min-width: 0;
  overflow: hidden;
`;

export const TitleRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const GroupField = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;
  min-width: 0;
`;

/** One-line code-style picker. Padded option labels share a decimal column. */
export const GroupSelect = styled.select`
  display: block;
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  margin: 0;
  padding: 6px 8px;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.sm}px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  color: ${({ theme }) => theme.semantic.text.primary};
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 12px;
  font-variant-numeric: tabular-nums lining-nums;
  line-height: 1.4;
`;

export const MetaRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

/** Markdown table: pipes + spaces, never a wrapping grid. */
export const BookPre = styled.pre`
  margin: 0;
  padding: ${({ theme }) => theme.spacing[2]}px;
  max-width: 100%;
  overflow: auto;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.sm}px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 11px;
  font-variant-numeric: tabular-nums lining-nums;
  line-height: 1.45;
  white-space: pre;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const BookLine = styled.div<{ $side?: 'bid' | 'ask'; $mid?: boolean }>`
  margin: 0;
  padding: 0;
  color: ${({ theme, $side, $mid }) =>
    $mid
      ? theme.semantic.text.secondary
      : $side === 'bid'
        ? theme.semantic.chart.up
        : $side === 'ask'
          ? theme.semantic.chart.down
          : theme.semantic.text.primary};
  font: inherit;
  font-variant-numeric: tabular-nums lining-nums;
  white-space: pre;
`;
