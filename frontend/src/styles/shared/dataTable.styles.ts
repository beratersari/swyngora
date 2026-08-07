import styled from 'styled-components';
import { Table } from 'antd';

/**
 * Shared dark trading-table chrome (Markets / Watchlist / Pumps / Alerts).
 * Single source of truth for header, hover, pagination, and card shell.
 */
export const DataTableCard = styled.div`
  background: ${({ theme }) => theme.semantic.bg.muted};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  overflow: hidden;
`;

export const DataTable = styled(Table)`
  .ant-table {
    background: transparent;
    color: ${({ theme }) => theme.semantic.text.primary};
  }

  .ant-table-thead > tr > th {
    background: ${({ theme }) => theme.semantic.bg.tableHeader ?? 'rgba(3, 98, 76, 0.45)'} !important;
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
    background: rgba(23, 135, 109, 0.28) !important;
  }

  .ant-table-tbody > tr.markets-row-clickable,
  .ant-table-tbody > tr[role='link'] {
    cursor: pointer;
  }

  .ant-table-placeholder .ant-table-cell {
    background: transparent !important;
  }

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

  .ant-pagination {
    margin: ${({ theme }) => theme.spacing[4]}px !important;
    color: ${({ theme }) => theme.semantic.text.primary};
  }

  .ant-pagination .ant-pagination-item {
    background: ${({ theme }) => theme.colors.pine};
    border-color: ${({ theme }) => theme.semantic.border.default};
  }

  .ant-pagination .ant-pagination-item a {
    color: ${({ theme }) => theme.semantic.text.primary};
  }

  .ant-pagination .ant-pagination-item-active {
    background: ${({ theme }) => theme.colors.bangladeshGreen};
    border-color: ${({ theme }) => theme.colors.caribbeanGreen};
  }

  .ant-pagination .ant-pagination-item-active a {
    color: ${({ theme }) => theme.colors.antiFlashWhite};
  }

  .ant-pagination .ant-pagination-prev .ant-pagination-item-link,
  .ant-pagination .ant-pagination-next .ant-pagination-item-link {
    background: ${({ theme }) => theme.colors.pine};
    border-color: ${({ theme }) => theme.semantic.border.default};
    color: ${({ theme }) => theme.semantic.text.primary};
  }

  .ant-pagination .ant-pagination-disabled .ant-pagination-item-link {
    opacity: 0.4;
  }

  .ant-pagination .ant-pagination-options .ant-select-selector {
    background: ${({ theme }) => theme.colors.pine} !important;
    border-color: ${({ theme }) => theme.semantic.border.default} !important;
    color: ${({ theme }) => theme.semantic.text.primary} !important;
  }

  .ant-pagination .ant-pagination-total-text {
    color: ${({ theme }) => theme.colors.pistachio};
    font-weight: 500;
  }

  .ant-spin-container::after {
    background: transparent;
  }
` as typeof Table;
