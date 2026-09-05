import styled from 'styled-components';

export const Stack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
`;

export const Head = styled.div`
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: ${({ theme }) => theme.spacing[2]}px;

  @media (max-width: 720px) {
    grid-template-columns: 1fr;
  }
`;

export const HeadCard = styled.div`
  min-width: 0;
  padding: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[3]}px;
  border-radius: ${({ theme }) => theme.radii.sm}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  background: ${({ theme }) => theme.semantic.bg.muted};
`;

export const Charts = styled.div`
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: ${({ theme }) => theme.spacing[3]}px;

  @media (max-width: 960px) {
    grid-template-columns: 1fr;
  }
`;

export const Chart = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
`;

export const Row = styled.div`
  display: grid;
  grid-template-columns: 88px minmax(0, 1fr) 56px;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: center;
`;

export const Track = styled.div`
  height: 12px;
  border-radius: 6px;
  background: ${({ theme }) => theme.semantic.bg.muted};
  overflow: hidden;
`;

export const Fill = styled.div<{ $width: number }>`
  height: 100%;
  width: ${({ $width }) => `${$width}%`};
  background: ${({ theme }) => theme.semantic.accent.default};
`;

export const Pair = styled.div`
  display: flex;
  align-items: flex-end;
  gap: 4px;
  height: 36px;
`;

export const PairBar = styled.div<{ $height: number; $side: 'call' | 'put' }>`
  width: 10px;
  height: ${({ $height }) => `${Math.max($height, 4)}%`};
  border-radius: 2px 2px 0 0;
  background: ${({ theme, $side }) =>
    $side === 'call' ? theme.semantic.chart.up : theme.semantic.chart.down};
`;
