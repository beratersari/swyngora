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
  padding: 0;
  text-align: right;
  background: ${({ $fill }) => $fill};
  border-top: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  color: ${({ theme }) => theme.semantic.text.primary};

  &:hover {
    filter: brightness(1.06);
  }
`;

export const CellButton = styled.button`
  display: block;
  width: 100%;
  margin: 0;
  padding: ${({ theme }) => theme.spacing[2]}px;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: right;
  cursor: pointer;

  &:focus-visible {
    outline: 2px solid ${({ theme }) => theme.semantic.border.focus};
    outline-offset: -2px;
  }
`;

export const DetailList = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const DetailRow = styled.div`
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;
