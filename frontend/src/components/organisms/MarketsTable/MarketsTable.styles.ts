import styled from 'styled-components';
import { Table } from 'antd';

export const TableCard = styled.div`
  background: ${({ theme }) => theme.semantic.bg.muted};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  overflow: hidden;
`;

export const StyledTable = styled(Table)`
  .ant-table {
    background: transparent;
    color: ${({ theme }) => theme.semantic.text.primary};
  }

  .ant-table-thead > tr > th {
    background: rgba(3, 98, 76, 0.45) !important;
    color: ${({ theme }) => theme.semantic.text.primary} !important;
    border-bottom-color: ${({ theme }) => theme.semantic.border.default} !important;
    font-weight: 600;
  }

  .ant-table-tbody > tr > td {
    border-bottom-color: ${({ theme }) => theme.semantic.border.default} !important;
    color: ${({ theme }) => theme.semantic.text.primary};
  }

  .ant-table-tbody > tr:hover > td {
    background: rgba(23, 135, 109, 0.28) !important;
  }

  .ant-table-tbody > tr.markets-row-clickable {
    cursor: pointer;
  }

  .ant-table-placeholder .ant-table-cell {
    background: transparent !important;
  }

  /* Pagination: high contrast on dark green */
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

export const TagList = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  max-width: 200px;
`;

export const EmptyWrap = styled.div`
  padding: ${({ theme }) => theme.spacing[6]}px;
  text-align: center;
`;

export const TableSkeletonWrap = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding: ${({ theme }) => theme.spacing[4]}px;
`;

export const SkeletonRow = styled.div`
  display: grid;
  grid-template-columns: 1.2fr 1fr 0.8fr 1fr 1fr 0.8fr 1.4fr;
  gap: ${({ theme }) => theme.spacing[3]}px;
  align-items: center;
`;
