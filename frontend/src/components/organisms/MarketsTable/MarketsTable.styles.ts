import styled from 'styled-components';
import { DataTable, DataTableCard } from '@/styles/shared/dataTable.styles';

export const TableCard = DataTableCard;
export const StyledTable = DataTable;

export const NameCell = styled.span`
  display: inline-flex;
  flex-direction: column;
  gap: 1px;
  line-height: 1.2;
`;

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
