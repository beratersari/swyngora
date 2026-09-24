import styled from 'styled-components';

export const Stack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
`;

export const Head = styled.div`
  display: grid;
  grid-template-columns: 72px 1fr 1fr 1fr;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: end;

  @media (max-width: 720px) {
    grid-template-columns: 64px 1fr;
  }
`;

export const Row = styled.div<{ $poc?: boolean }>`
  display: grid;
  grid-template-columns: 72px 1fr 1fr 1fr;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: center;
  padding: 2px 0;
  border-radius: ${({ theme }) => theme.radii.sm}px;
  background: ${({ theme, $poc }) => ($poc ? theme.semantic.bg.muted : 'transparent')};

  @media (max-width: 720px) {
    grid-template-columns: 64px 1fr;
  }
`;

export const Track = styled.div`
  display: flex;
  height: 10px;
  min-width: 0;
  border-radius: 999px;
  overflow: hidden;
  background: ${({ theme }) => theme.semantic.bg.muted};
`;

export const Seg = styled.span<{ $pct: number; $side: 'up' | 'down' }>`
  width: ${({ $pct }) => `${Math.max($pct, 0)}%`};
  background: ${({ theme, $side }) =>
    $side === 'up' ? theme.semantic.chart.up : theme.semantic.chart.down};
`;

export const Legend = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[3]}px;
`;

export const Swatch = styled.span<{ $side: 'up' | 'down' }>`
  display: inline-block;
  width: 8px;
  height: 8px;
  margin-right: 4px;
  border-radius: 2px;
  background: ${({ theme, $side }) =>
    $side === 'up' ? theme.semantic.chart.up : theme.semantic.chart.down};
`;
