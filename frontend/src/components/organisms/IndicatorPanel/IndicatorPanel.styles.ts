import styled from 'styled-components';

export const Panel = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  padding: ${({ theme }) => theme.spacing[4]}px;
  background: ${({ theme }) => theme.semantic.bg.muted};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.lg}px;
`;

export const PanelHead = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const SnapshotGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const SnapshotCard = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: ${({ theme }) => theme.spacing[3]}px;
  background: ${({ theme }) => theme.semantic.bg.chrome};
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  border-radius: ${({ theme }) => theme.radii.md}px;
`;

export const ChartBlock = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const LegendRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[3]}px;
  align-items: center;
`;

export const LegendSwatch = styled.span<{ $color: string }>`
  display: inline-block;
  width: 12px;
  height: 3px;
  border-radius: 2px;
  background: ${({ $color }) => $color};
  margin-right: 6px;
  vertical-align: middle;
`;
