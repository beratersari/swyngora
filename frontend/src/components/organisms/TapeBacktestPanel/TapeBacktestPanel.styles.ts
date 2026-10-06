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
  flex: 1 1 140px;
`;

export const VenueBlock = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const PerfGrid = styled.div`
  display: grid;
  grid-template-columns: minmax(88px, 1.2fr) repeat(3, minmax(64px, 1fr));
  gap: 6px 10px;
  align-items: baseline;
`;
