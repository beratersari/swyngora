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
  max-width: 100%;
  min-width: 0;

  /* Let Ant table horizontal scroll stay inside the card on narrow viewports */
  .ant-table-wrapper,
  .ant-spin-nested-loading,
  .ant-spin-container {
    min-width: 0;
    max-width: 100%;
  }
`;

export const DataTable = styled(Table)`
  .ant-table {
    background: transparent;
    color: ${({ theme }) => theme.semantic.text.primary};
  }

  .ant-table-thead > tr > th {
    background: ${({ theme }) => theme.semantic.bg.tableHeader} !important;
    color: ${({ theme }) => theme.semantic.text.secondary} !important;
    border-bottom: 1px solid ${({ theme }) => theme.semantic.border.subtle} !important;
    font-weight: 600;
    font-size: 11px;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    padding: 8px 12px !important;
  }

  .ant-table-tbody > tr > td {
    border-bottom-color: ${({ theme }) => theme.semantic.border.subtle} !important;
    color: ${({ theme }) => theme.semantic.text.primary};
    background: transparent !important;
    padding: 7px 12px !important;
    transition: background-color ${({ theme }) => theme.motion.duration.fast}
      ${({ theme }) => theme.motion.ease.standard};
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
    flex-wrap: wrap;
    row-gap: 8px;
  }

  .ant-pagination .ant-pagination-item {
    background: ${({ theme }) => theme.colors.pine};
    border-color: ${({ theme }) => theme.semantic.border.default};
  }

  .ant-pagination .ant-pagination-item a {
    color: ${({ theme }) => theme.semantic.text.primary};
  }

  .ant-pagination .ant-pagination-item-active {
    background: ${({ theme }) => theme.semantic.action.primary};
    border-color: ${({ theme }) => theme.semantic.border.accent};
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
    color: ${({ theme }) => theme.semantic.text.secondary};
    font-weight: 500;
  }

  .ant-spin-container::after {
    background: transparent;
  }
` as typeof Table;
