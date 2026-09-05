import styled from 'styled-components';

export const Stack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
`;

export const Row = styled.div`
  display: grid;
  grid-template-columns: 112px minmax(0, 1fr) 72px;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: center;
`;

export const Track = styled.div`
  position: relative;
  height: 14px;
  border-radius: 7px;
  background: ${({ theme }) => theme.semantic.bg.muted};
  overflow: hidden;
`;

export const Fill = styled.div<{ $width: number }>`
  height: 100%;
  width: ${({ $width }) => `${$width}%`};
  display: flex;
  background: ${({ theme }) => theme.semantic.bg.accentMuted};
`;

export const CallSlice = styled.div<{ $share: number }>`
  height: 100%;
  width: ${({ $share }) => `${$share * 100}%`};
  background: ${({ theme }) => theme.semantic.chart.up};
`;

export const PutSlice = styled.div`
  flex: 1;
  height: 100%;
  background: ${({ theme }) => theme.semantic.chart.down};
`;
