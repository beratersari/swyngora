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

export const Profile = styled.div`
  display: flex;
  align-items: stretch;
  gap: 2px;
  height: 88px;
  min-width: 0;
`;

export const Col = styled.div<{ $spot?: boolean; $flip?: boolean }>`
  flex: 1 1 0;
  min-width: 2px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  outline: ${({ theme, $spot, $flip }) =>
    $flip
      ? `1px solid ${theme.semantic.accent.default}`
      : $spot
        ? `1px dashed ${theme.semantic.border.strong}`
        : 'none'};
  outline-offset: -1px;
`;

export const Up = styled.div`
  flex: 1 1 0;
  display: flex;
  align-items: flex-end;
`;

export const Down = styled.div`
  flex: 1 1 0;
  display: flex;
  align-items: flex-start;
`;

export const Bar = styled.div<{ $height: number; $side: 'long' | 'short' }>`
  width: 100%;
  height: ${({ $height }) => `${Math.max($height, 0)}%`};
  min-height: ${({ $height }) => ($height > 0 ? '2px' : '0')};
  background: ${({ theme, $side }) =>
    $side === 'long' ? theme.semantic.chart.up : theme.semantic.chart.down};
`;

export const Axis = styled.div`
  height: 1px;
  background: ${({ theme }) => theme.semantic.border.subtle};
`;
