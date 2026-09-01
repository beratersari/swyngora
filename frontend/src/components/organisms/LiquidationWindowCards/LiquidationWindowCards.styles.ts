import styled from 'styled-components';

export const Grid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 260px;

  @media (max-width: 720px) {
    grid-template-columns: 1fr 1fr;
    min-width: 0;
  }
`;

export const CardButton = styled.button<{ $active: boolean }>`
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 6px;
  margin: 0;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid
    ${({ theme, $active }) =>
      $active ? theme.semantic.border.focus : theme.semantic.border.default};
  background: ${({ theme }) => theme.semantic.bg.canvas};
  box-shadow: ${({ $active }) =>
    $active ? '0 0 0 1px rgba(56, 97, 251, 0.28)' : 'none'};
  cursor: pointer;
  text-align: left;
  color: inherit;

  &:hover {
    border-color: ${({ theme }) => theme.semantic.border.strong};
  }

  &:focus-visible {
    outline: 2px solid ${({ theme }) => theme.semantic.border.focus};
    outline-offset: 2px;
  }
`;

export const CardHead = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
`;

export const WindowLabel = styled.span`
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: ${({ theme }) => theme.semantic.text.tertiary};
`;

export const TotalValue = styled.span`
  font-family: ${({ theme }) => theme.fontFamilies.sans};
  font-variant-numeric: tabular-nums;
  font-size: 22px;
  font-weight: 700;
  line-height: 1.15;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const SideRow = styled.div`
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
  font-size: 13px;
  font-variant-numeric: tabular-nums;
`;

export const SideLabel = styled.span<{ $tone: 'long' | 'short' }>`
  font-weight: 600;
  color: ${({ theme, $tone }) =>
    $tone === 'long' ? theme.semantic.chart.down : theme.semantic.chart.up};
`;

export const SideValue = styled.span<{ $tone: 'long' | 'short' }>`
  font-weight: 700;
  color: ${({ theme, $tone }) =>
    $tone === 'long' ? theme.semantic.chart.down : theme.semantic.chart.up};
`;

export const SplitBar = styled.div`
  display: flex;
  height: 6px;
  border-radius: 99px;
  overflow: hidden;
  background: ${({ theme }) => theme.semantic.bg.muted};
`;

export const SplitLong = styled.span<{ $pct: number }>`
  width: ${({ $pct }) => $pct}%;
  background: ${({ theme }) => theme.semantic.chart.down};
`;

export const SplitShort = styled.span<{ $pct: number }>`
  width: ${({ $pct }) => $pct}%;
  background: ${({ theme }) => theme.semantic.chart.up};
`;
