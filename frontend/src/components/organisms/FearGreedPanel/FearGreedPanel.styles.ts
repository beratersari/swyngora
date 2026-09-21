import styled from 'styled-components';

export const ModalTitle = styled.span`
  display: inline-flex;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const InfoButton = styled.button`
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  margin: 0;
  padding: 0;
  border: 0;
  border-radius: ${({ theme }) => theme.radii.sm}px;
  background: transparent;
  color: ${({ theme }) => theme.semantic.text.secondary};
  cursor: pointer;

  &:hover {
    background: ${({ theme }) => theme.semantic.bg.muted};
    color: ${({ theme }) => theme.semantic.text.primary};
  }
`;

export const FactorList = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const FactorRow = styled.div`
  display: flex;
  flex-direction: column;
  gap: 2px;
`;

export const FactorHead = styled.div`
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

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
