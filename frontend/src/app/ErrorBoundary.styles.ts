import styled from 'styled-components';

export const ErrorBoundaryShell = styled.div`
  min-height: 40vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: ${({ theme }) => theme.spacing[6]}px;
`;
