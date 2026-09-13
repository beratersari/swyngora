import styled from 'styled-components';

export const Detail = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
  padding: ${({ theme }) => theme.spacing[2]}px 0;
`;
