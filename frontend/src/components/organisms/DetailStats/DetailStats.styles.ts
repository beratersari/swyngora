import styled from 'styled-components';

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

export const StatsSection = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;
