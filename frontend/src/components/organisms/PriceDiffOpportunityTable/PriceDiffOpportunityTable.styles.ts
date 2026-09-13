import styled from 'styled-components';

export const Review = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
  padding: ${({ theme }) => theme.spacing[2]}px 0;
`;

export const Spark = styled.svg`
  display: block;
  width: 160px;
  height: 36px;
`;
