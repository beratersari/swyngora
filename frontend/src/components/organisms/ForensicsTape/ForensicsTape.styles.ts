import styled from 'styled-components';

export const Stack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[4]}px;
  min-width: 0;
`;

export const PhaseGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: ${({ theme }) => theme.spacing[3]}px;
  @media (max-width: 960px) {
    grid-template-columns: 1fr;
  }
`;

export const PhaseCard = styled.section<{ $incomplete?: boolean }>`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
  padding: ${({ theme }) => theme.spacing[3]}px;
  background: ${({ theme }) => theme.semantic.bg.elevated};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  opacity: ${({ $incomplete }) => ($incomplete ? 0.72 : 1)};
`;

export const PhaseHead = styled.div`
  display: flex;
  flex-direction: column;
  gap: 2px;
`;

export const MetricGrid = styled.dl`
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 6px 12px;
  margin: 0;
`;

export const MetricLabel = styled.dt`
  margin: 0;
`;

export const MetricValue = styled.dd`
  margin: 0;
  text-align: right;
  font-variant-numeric: tabular-nums;
`;

export const ChangeTable = styled.div`
  overflow-x: auto;
`;
