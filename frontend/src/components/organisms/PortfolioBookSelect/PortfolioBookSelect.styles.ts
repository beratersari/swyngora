import styled from 'styled-components';

export const Row = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[3]}px;
  align-items: flex-end;
  min-width: 0;
  max-width: 100%;

  ${({ theme }) => theme.media.phone} {
    flex-direction: column;
    align-items: stretch;
    gap: ${({ theme }) => theme.spacing[2]}px;

    .ant-btn {
      width: 100%;
    }
  }
`;

export const Field = styled.label`
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  flex: 1 1 200px;

  ${({ theme }) => theme.media.phone} {
    flex: 1 1 auto;
    width: 100%;
  }

  .ant-select {
    width: 100%;
    min-width: 0;
  }
`;
