import styled from 'styled-components';

export const Panel = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const TitleRow = styled.div`
  display: flex;
  flex-direction: column;
  gap: 2px;
`;

export const StatsGrid = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[1]}px ${({ theme }) => theme.spacing[5]}px;
  padding: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[3]}px;
  background: ${({ theme }) => theme.semantic.bg.muted};
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  border-radius: ${({ theme }) => theme.radii.md}px;
`;

export const StatCard = styled.div`
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 88px;
  padding: ${({ theme }) => theme.spacing[1]}px 0;
`;

export const AddressWrap = styled.div`
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
`;

export const AddressLabel = styled.span`
  font-size: ${({ theme }) => theme.typeScale.caption.fontSize}px;
  font-weight: ${({ theme }) => theme.fontWeights.semibold};
  color: ${({ theme }) => theme.semantic.text.primary};
  line-height: 1.2;
`;

export const AddressCell = styled.span`
  font-variant-numeric: tabular-nums;
`;
