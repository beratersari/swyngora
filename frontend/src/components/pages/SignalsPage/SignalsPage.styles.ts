import styled from 'styled-components';

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[5]}px;
  min-width: 0;
`;

export const Section = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const MetricStrip = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[5]}px;
  padding: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[3]}px;
  background: ${({ theme }) => theme.semantic.bg.muted};
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  border-radius: ${({ theme }) => theme.radii.md}px;
`;

export const MetricCard = styled.div`
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 96px;
  padding: ${({ theme }) => theme.spacing[1]}px 0;
  background: transparent;
  border: 0;
  border-radius: 0;
`;

export const DeskSplit = styled.div`
  display: grid;
  grid-template-columns: minmax(280px, 1fr) minmax(320px, 1.1fr);
  gap: ${({ theme }) => theme.spacing[5]}px;

  ${({ theme }) => theme.media.tablet} {
    grid-template-columns: 1fr;
  }
`;
