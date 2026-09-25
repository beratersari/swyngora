import styled from 'styled-components';

export const Stack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
`;

export const Head = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const List = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 420px;
  overflow: auto;
`;

export const Row = styled.div`
  display: grid;
  grid-template-columns: 88px minmax(0, 1fr) minmax(0, 1fr);
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: center;

  @media (max-width: 720px) {
    grid-template-columns: 72px minmax(0, 1fr);
  }
`;

export const Side = styled.div`
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 6px;
  align-items: center;
  min-width: 0;
`;

export const Track = styled.div`
  height: 8px;
  border-radius: 4px;
  background: ${({ theme }) => theme.semantic.bg.muted};
  overflow: hidden;
`;

export const Fill = styled.div<{ $width: number; $up: boolean }>`
  height: 100%;
  width: ${({ $width }) => `${$width}%`};
  background: ${({ theme, $up }) => ($up ? theme.semantic.chart.up : theme.semantic.chart.down)};
`;
