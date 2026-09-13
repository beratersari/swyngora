import styled from 'styled-components';

export const Card = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
  padding: ${({ theme }) => theme.spacing[4]}px;
  background: ${({ theme }) => theme.semantic.bg.elevated};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  border-left: 3px solid ${({ theme }) => theme.semantic.accent.default};
`;

export const Sections = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const Section = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
`;
