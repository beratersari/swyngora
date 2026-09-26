import styled from 'styled-components';

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[4]}px;
  min-width: 0;
`;

export const Toolbar = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[3]}px;
  align-items: flex-end;
`;

export const Field = styled.label`
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

export const WindowGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const WindowCard = styled.button<{ $active?: boolean }>`
  text-align: left;
  border: 1px solid
    ${({ theme, $active }) => ($active ? theme.semantic.border.strong : theme.semantic.border.subtle)};
  background: ${({ theme }) => theme.semantic.bg.elevated};
  border-radius: ${({ theme }) => theme.radii.md}px;
  padding: ${({ theme }) => theme.spacing[3]}px;
  cursor: pointer;
  color: inherit;
`;
