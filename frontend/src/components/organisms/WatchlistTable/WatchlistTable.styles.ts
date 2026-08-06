import styled from 'styled-components';
import { Table } from 'antd';

export const TableCard = styled.div`
  background: ${({ theme }) => theme.semantic.bg.muted};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  overflow: hidden;
`;

/** Dark-theme table — avoid default Ant white chrome under cells/actions. */
export const StyledTable = styled(Table)`
  .ant-table {
    background: transparent;
    color: ${({ theme }) => theme.semantic.text.primary};
  }

  .ant-table-thead > tr > th {
    background: ${({ theme }) => theme.semantic.bg.tableHeader} !important;
    color: ${({ theme }) => theme.semantic.text.primary} !important;
    border-bottom-color: ${({ theme }) => theme.semantic.border.default} !important;
    font-weight: 600;
  }

  .ant-table-tbody > tr > td {
    border-bottom-color: ${({ theme }) => theme.semantic.border.default} !important;
    color: ${({ theme }) => theme.semantic.text.primary};
    background: transparent !important;
  }

  .ant-table-tbody > tr:hover > td {
    background: ${({ theme }) => theme.semantic.bg.hover} !important;
  }

  .ant-table-placeholder .ant-table-cell {
    background: transparent !important;
  }

  /* scroll.x measure / sticky bar often renders as a pale strip under the last column */
  .ant-table-measure-row,
  .ant-table-measure-row td {
    visibility: collapse !important;
    height: 0 !important;
    padding: 0 !important;
    border: none !important;
    line-height: 0 !important;
  }

  .ant-table-sticky-scroll,
  .ant-table-sticky-scroll-bar {
    display: none !important;
  }

  .ant-table-body {
    overflow-x: auto !important;
  }

  .ant-spin-container::after {
    background: transparent;
  }
` as typeof Table;
