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

export const Groups = styled.div`
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  padding-left: ${({ theme }) => theme.spacing[3]}px;
`;

export const OpenButton = styled.button`
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  margin: 0;
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
  min-width: 0;

  &:focus-visible {
    outline: 1px solid currentColor;
    outline-offset: 2px;
  }
`;

export const CloseList = styled.ul`
  list-style: none;
  margin: 0;
  padding: 0 0 0 ${({ theme }) => theme.spacing[3]}px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
`;
