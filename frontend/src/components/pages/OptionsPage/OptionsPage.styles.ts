import styled from 'styled-components';

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
  width: 100%;
`;

export const Toolbar = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const Panel = styled.section`
  min-width: 0;
  padding: ${({ theme }) => theme.spacing[3]}px;
  border-radius: ${({ theme }) => theme.radii.md}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  background: ${({ theme }) => theme.semantic.bg.elevated};
`;
