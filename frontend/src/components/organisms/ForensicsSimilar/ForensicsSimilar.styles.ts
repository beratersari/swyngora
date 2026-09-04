import styled from 'styled-components';

export const Stack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
`;

export const TableWrap = styled.div`
  overflow-x: auto;
`;
