import styled from 'styled-components';

export const Layout = styled.div`
  display: grid;
  grid-template-columns: minmax(280px, 340px) minmax(0, 1fr);
  gap: ${({ theme }) => theme.spacing[5]}px;

  ${({ theme }) => theme.media.tablet} {
    grid-template-columns: 1fr;
  }
`;

export const Mixer = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
`;

export const FactorList = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const FactorRow = styled.label`
  display: grid;
  grid-template-columns: 22px minmax(0, 1fr) 88px;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const Results = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
`;

export const Card = styled.article`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding: ${({ theme }) => theme.spacing[4]}px;
  background: ${({ theme }) => theme.semantic.bg.muted};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  min-width: 0;
`;

export const CardTop = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const FactorBreakdown = styled.ul`
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

export const FactorLine = styled.li`
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: baseline;
`;

export const ActionRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;
