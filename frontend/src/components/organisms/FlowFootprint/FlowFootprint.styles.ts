import styled from 'styled-components';

export const Stack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
`;

export const Legend = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[3]}px;
`;

export const Swatch = styled.span<{ $side: 'up' | 'down' | 'poc' }>`
  display: inline-block;
  width: 8px;
  height: 8px;
  margin-right: 4px;
  border-radius: 2px;
  background: ${({ theme, $side }) => {
    if ($side === 'poc') return theme.semantic.accent.default;
    return $side === 'up' ? theme.semantic.chart.up : theme.semantic.chart.down;
  }};
`;

export const Head = styled.div`
  display: grid;
  grid-template-columns: 1fr 88px 1fr 72px;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: end;
`;

export const Scroller = styled.div`
  max-height: 420px;
  overflow: auto;
  min-width: 0;
`;

export const Row = styled.div<{ $poc?: boolean; $side?: 'buy' | 'sell' | null }>`
  display: grid;
  grid-template-columns: 1fr 88px 1fr 72px;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: center;
  padding: 3px 4px;
  border-radius: ${({ theme }) => theme.radii.sm}px;
  background: ${({ theme, $poc, $side }) => {
    if ($poc) return theme.semantic.bg.accentSoft;
    if ($side === 'buy') return 'rgba(22, 199, 132, 0.12)';
    if ($side === 'sell') return 'rgba(234, 57, 67, 0.12)';
    return 'transparent';
  }};
  box-shadow: ${({ theme, $poc }) => ($poc ? `inset 2px 0 0 ${theme.semantic.accent.default}` : 'none')};
`;

export const Side = styled.div<{ $align: 'start' | 'end' }>`
  display: flex;
  flex-direction: ${({ $align }) => ($align === 'end' ? 'row-reverse' : 'row')};
  align-items: center;
  gap: 6px;
  min-width: 0;
`;

export const Track = styled.div`
  flex: 1;
  min-width: 0;
  height: 10px;
  border-radius: 999px;
  background: ${({ theme }) => theme.semantic.bg.muted};
  overflow: hidden;
  display: flex;
`;

export const TrackEnd = styled(Track)`
  justify-content: flex-end;
`;

export const Seg = styled.span<{ $pct: number; $side: 'up' | 'down' }>`
  width: ${({ $pct }) => `${$pct}%`};
  height: 100%;
  background: ${({ theme, $side }) =>
    $side === 'up' ? theme.semantic.chart.up : theme.semantic.chart.down};
`;

export const PriceCell = styled.div`
  display: flex;
  flex-direction: column;
  min-width: 0;
`;

export const Badge = styled.span<{ $kind: 'poc' | 'buy' | 'sell' }>`
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: ${({ theme, $kind }) => {
    if ($kind === 'poc') return theme.semantic.accent.default;
    return $kind === 'buy' ? theme.semantic.chart.up : theme.semantic.chart.down;
  }};
`;
