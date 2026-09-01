import styled from 'styled-components';

export const Strip = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
`;

export const VenueCard = styled.div<{ $live: boolean }>`
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 180px;
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid
    ${({ theme, $live }) =>
      $live ? theme.semantic.border.default : theme.semantic.chart.down};
  background: ${({ theme }) => theme.semantic.bg.canvas};
`;

export const VenueHead = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
`;

export const LiveDot = styled.span<{ $live: boolean }>`
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: ${({ $live, theme }) =>
    $live ? theme.semantic.chart.up : theme.semantic.chart.down};
`;

export const Meta = styled.div`
  display: flex;
  flex-direction: column;
  gap: 1px;
  font-size: 11px;
  color: ${({ theme }) => theme.semantic.text.secondary};
  font-variant-numeric: tabular-nums;
`;