import styled from 'styled-components';

export const List = styled.ul`
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
`;

export const Row = styled.li`
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
`;
