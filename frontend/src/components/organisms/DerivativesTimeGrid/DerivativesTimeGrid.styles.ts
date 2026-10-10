import { Link } from 'react-router-dom';
import styled from 'styled-components';

export const Scroll = styled.div`
  overflow-x: auto;
  min-width: 0;
`;

export const Table = styled.table`
  width: max-content;
  min-width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  font-variant-numeric: tabular-nums;
`;

export const HeadCell = styled.th`
  position: sticky;
  top: 0;
  padding: ${({ theme }) => theme.spacing[2]}px;
  text-align: right;
  font-weight: ${({ theme }) => theme.fontWeights.medium};
  color: ${({ theme }) => theme.semantic.text.secondary};
  background: ${({ theme }) => theme.semantic.bg.canvas};
  white-space: nowrap;

  &:first-child {
    left: 0;
    z-index: 2;
    text-align: left;
  }
`;

export const CoinCell = styled.td`
  position: sticky;
  left: 0;
  z-index: 1;
  padding: ${({ theme }) => theme.spacing[2]}px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border-top: 1px solid ${({ theme }) => theme.semantic.border.subtle};
`;

export const CoinLink = styled(Link)`
  color: ${({ theme }) => theme.semantic.text.primary};
  font-weight: ${({ theme }) => theme.fontWeights.medium};
  text-decoration: none;
`;

export const Cell = styled.td<{ $fill: string }>`
  min-width: 72px;
  padding: ${({ theme }) => theme.spacing[2]}px;
  text-align: right;
  background: ${({ $fill }) => $fill};
  border-top: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  color: ${({ theme }) => theme.semantic.text.primary};
`;
