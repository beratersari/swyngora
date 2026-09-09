import styled from 'styled-components';

export const Stack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[4]}px;
`;

export const FieldRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[3]}px;
  align-items: flex-end;
`;

export const Field = styled.label`
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 140px;
`;

export const SignalDrawerBody = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const FactorWhy = styled.div`
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 4px 0;
`;

export const PerfTable = styled.div`
  display: flex;
  flex-direction: column;
  gap: 8px;
`;

export const PerfGrid = styled.div`
  display: grid;
  grid-template-columns: minmax(110px, 1.2fr) repeat(3, minmax(64px, 1fr));
  gap: 6px 10px;
  align-items: baseline;
`;

export const FactorLine = styled.div`
  display: grid;
  grid-template-columns: minmax(120px, 1.4fr) 72px 64px 72px;
  gap: 8px;
  align-items: baseline;
`;
