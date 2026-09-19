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
  align-items: flex-end;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const Field = styled.label`
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 168px;
`;
