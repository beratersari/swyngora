import styled from 'styled-components';

export const ChartShell = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
  max-width: 100%;
`;

export const ChartToolbar = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;

  ${({ theme }) => theme.media.phone} {
    flex-direction: column;
    align-items: stretch;

    .ant-segmented {
      width: 100%;
      overflow-x: auto;
    }

    .ant-segmented-group {
      width: 100%;
      display: flex;
    }

    .ant-segmented-item {
      flex: 1 1 0;
      min-width: 0;
    }
  }
`;

export const ChartBox = styled.div`
  width: 100%;
  min-width: 0;
  min-height: 180px;
  border-radius: ${({ theme }) => theme.radii.md}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  background: ${({ theme }) => theme.semantic.bg.canvas};
  overflow: hidden;
`;
