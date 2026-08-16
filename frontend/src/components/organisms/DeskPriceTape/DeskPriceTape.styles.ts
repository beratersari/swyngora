import styled from 'styled-components';

export const Bar = styled.div`
  display: flex;
  align-items: stretch;
  height: 36px;
  overflow: hidden;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border-bottom: 1px solid ${({ theme }) => theme.semantic.border.default};
`;

export const SourceGroup = styled.div`
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 2px;
  padding: 4px 8px 4px 16px;
  border-right: 1px solid ${({ theme }) => theme.semantic.border.subtle};
`;

export const SourceBtn = styled.button<{ $active?: boolean }>`
  appearance: none;
  border: 0;
  background: ${({ theme, $active }) => ($active ? theme.semantic.bg.hover : 'transparent')};
  color: ${({ theme, $active }) =>
    $active ? theme.semantic.accent.default : theme.semantic.text.secondary};
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.02em;
  text-transform: uppercase;
  line-height: 1;
  padding: 7px 8px;
  border-radius: 6px;
  cursor: pointer;
  white-space: nowrap;

  &:hover {
    color: ${({ theme }) => theme.semantic.accent.default};
    background: ${({ theme }) => theme.semantic.bg.hover};
  }
`;

export const TapeSlot = styled.div`
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
`;

export const Empty = styled.span`
  padding: 0 16px;
  font-size: 12px;
  color: ${({ theme }) => theme.semantic.text.tertiary};
  white-space: nowrap;
`;
