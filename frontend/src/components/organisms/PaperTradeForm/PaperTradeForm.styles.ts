import styled from 'styled-components';

export const FormStack = styled.div<{ $compact?: boolean }>`
  display: flex;
  flex-direction: column;
  gap: ${({ theme, $compact }) => theme.spacing[$compact ? 2 : 3]}px;
  min-width: 0;
  max-width: 100%;
`;

export const FieldRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[3]}px;
  align-items: flex-end;
  min-width: 0;

  ${({ theme }) => theme.media.phone} {
    flex-direction: column;
    align-items: stretch;
    gap: ${({ theme }) => theme.spacing[2]}px;
  }
`;

export const Field = styled.label`
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  flex: 1 1 140px;

  ${({ theme }) => theme.media.phone} {
    flex: 1 1 auto;
    width: 100%;
  }

  /* Ant controls fill the field on narrow layouts */
  .ant-select,
  .ant-input-number,
  .ant-input,
  .ant-segmented {
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
    flex-direction: column;
    align-items: stretch;

    .ant-btn {
      width: 100%;
    }
  }
`;
