import styled from 'styled-components';

export const Stack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const Read = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[1]}px;
`;
