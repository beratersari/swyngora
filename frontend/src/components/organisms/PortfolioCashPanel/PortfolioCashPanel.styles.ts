import styled from 'styled-components';

export const Panel = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
  max-width: 100%;
`;

export const FieldRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: flex-end;
  min-width: 0;

  ${({ theme }) => theme.media.phone} {
    flex-direction: column;
    align-items: stretch;
  }
`;

export const Field = styled.label`
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  flex: 1 1 120px;

  ${({ theme }) => theme.media.phone} {
    flex: 1 1 auto;
    width: 100%;
  }

  .ant-input-number,
  .ant-input {
    width: 100%;
    min-width: 0;
  }
`;

export const Actions = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: center;

  ${({ theme }) => theme.media.phone} {
    width: 100%;

    .ant-btn {
      flex: 1 1 auto;
    }
  }
`;
