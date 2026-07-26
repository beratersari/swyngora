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
  }

  .ant-table-thead > tr > th {
    background: rgba(75, 86, 148, 0.35) !important;
    color: ${({ theme }) => theme.semantic.text.primary};
    border-bottom-color: ${({ theme }) => theme.semantic.border.default} !important;
  }

  .ant-table-tbody > tr > td {
    border-bottom-color: ${({ theme }) => theme.semantic.border.default} !important;
  }

  .ant-table-tbody > tr:hover > td {
    background: rgba(75, 86, 148, 0.2) !important;
  }

  .ant-pagination {
    margin: ${({ theme }) => theme.spacing[4]}px !important;
  }
` as typeof Table;

export const TagList = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  max-width: 200px;
`;

export const EmptyWrap = styled.div`
  padding: ${({ theme }) => theme.spacing[8]}px;
  text-align: center;
`;
