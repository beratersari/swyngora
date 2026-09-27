import styled from 'styled-components';

export const DetailList = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const DetailRow = styled.div`
  display: flex;
  flex-direction: column;
  gap: 2px;
`;

export const ChartBox = styled.div`
  width: 100%;
  height: 120px;
`;

export const Head = styled.div`
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;
