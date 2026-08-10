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
`;

export const TitleRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const MetaRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const Ladder = styled.div`
  display: flex;
  flex-direction: column;
  font-variant-numeric: tabular-nums;
  font-size: 12px;
`;

export const Head = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding: 0 0 ${({ theme }) => theme.spacing[1]}px;
  color: ${({ theme }) => theme.semantic.text.secondary};
`;

export const Row = styled.button<{ $side: 'bid' | 'ask'; $wall?: boolean }>`
  position: relative;
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: ${({ theme }) => theme.spacing[2]}px;
  width: 100%;
  margin: 0;
  padding: 3px 0;
  border: 0;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: default;
  font: inherit;
  font-variant-numeric: tabular-nums;
  ${({ $wall, theme }) =>
    $wall ? `box-shadow: inset 0 0 0 1px ${theme.semantic.border.default};` : ''}
`;

export const DepthBar = styled.span<{ $side: 'bid' | 'ask'; $pct: number }>`
  position: absolute;
  top: 1px;
  bottom: 1px;
  right: 0;
  width: ${({ $pct }) => `${$pct}%`};
  background: ${({ theme, $side }) =>
    $side === 'bid'
      ? theme.semantic.chart.up
      : theme.semantic.chart.down};
  opacity: 0.16;
  pointer-events: none;
`;

export const Price = styled.span<{ $side: 'bid' | 'ask' }>`
  position: relative;
  z-index: 1;
  color: ${({ theme, $side }) =>
    $side === 'bid' ? theme.semantic.chart.up : theme.semantic.chart.down};
  font-weight: ${({ theme }) => theme.fontWeights.medium};
`;

export const Qty = styled.span`
  position: relative;
  z-index: 1;
  text-align: right;
`;

export const SpreadRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding: ${({ theme }) => theme.spacing[2]}px 0;
  margin: ${({ theme }) => theme.spacing[1]}px 0;
  border-top: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-bottom: 1px solid ${({ theme }) => theme.semantic.border.default};
`;

export const WallTag = styled.span`
  position: relative;
  z-index: 1;
  margin-left: 6px;
  font-size: 10px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: ${({ theme }) => theme.semantic.text.secondary};
`;
