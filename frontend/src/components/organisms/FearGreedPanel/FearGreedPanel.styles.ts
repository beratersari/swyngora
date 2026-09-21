import styled from 'styled-components';

export const HistoryRow = styled.div`
  display: flex;
  align-items: flex-end;
  gap: 2px;
  height: 36px;
  width: 100%;
`;

export const HistoryBar = styled.span<{ $height: number; $tone: 'up' | 'down' | 'flat' }>`
  flex: 1 1 0;
  min-width: 2px;
  max-width: 8px;
  height: ${({ $height }) => Math.max(3, $height)}px;
  border-radius: 1px;
  background: ${({ theme, $tone }) => {
    if ($tone === 'up') return theme.semantic.status.success;
    if ($tone === 'down') return theme.semantic.status.error;
    return theme.semantic.chart.neutral;
  }};
`;
