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

export const WhyCell = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[1]}px;
  min-width: 220px;
  max-width: 420px;
`;

export const FieldScores = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[1]}px ${({ theme }) => theme.spacing[2]}px;
`;
