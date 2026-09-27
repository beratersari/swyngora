import styled from 'styled-components';

export const Card = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
`;

export const Stats = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[4]}px;
`;

export const GapList = styled.ul`
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;
