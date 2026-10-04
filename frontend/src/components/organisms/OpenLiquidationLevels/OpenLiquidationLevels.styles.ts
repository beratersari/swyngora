import styled from 'styled-components';

export const Panel = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
`;

export const Summary = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[3]}px ${({ theme }) => theme.spacing[4]}px;
`;

export const TableWrap = styled.div`
  overflow-x: auto;
  min-width: 0;
`;

export const Table = styled.table`
  width: 100%;
  border-collapse: collapse;
  font-family: ${({ theme }) => theme.fontFamilies.sans};
  font-size: 13px;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const Th = styled.th`
  text-align: left;
  padding: 8px 10px;
  border-bottom: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  color: ${({ theme }) => theme.semantic.text.secondary};
  font-weight: 500;
  white-space: nowrap;
`;

export const Td = styled.td`
  padding: 8px 10px;
  border-bottom: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  white-space: nowrap;
`;
