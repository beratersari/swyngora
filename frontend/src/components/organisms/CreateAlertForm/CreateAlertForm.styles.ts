import styled from 'styled-components';

export const FormStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  max-width: 480px;
  width: 100%;
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
  min-width: 120px;
  flex: 1 1 140px;

  ${({ theme }) => theme.media.phone} {
    flex: 1 1 100%;
    min-width: 0;
  }
`;
