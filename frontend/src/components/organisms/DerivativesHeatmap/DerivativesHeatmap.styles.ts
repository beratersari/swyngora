import { Link } from 'react-router-dom';
import styled from 'styled-components';

export const Grid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(112px, 1fr));
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
`;

export const Cell = styled(Link)`
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 4px;
  min-height: 72px;
  min-width: 0;
  padding: ${({ theme }) => theme.spacing[2]}px;
  border-radius: ${({ theme }) => theme.radii.md}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.strong};
  color: ${({ theme }) => theme.semantic.text.primary};
  text-decoration: none;
`;

export const Value = styled.span`
  font-variant-numeric: tabular-nums;
  word-break: break-word;
`;
