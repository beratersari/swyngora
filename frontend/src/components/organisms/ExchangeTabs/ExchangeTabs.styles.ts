import styled from 'styled-components';

export const TabList = styled.div`
  display: inline-flex;
  align-items: center;
  gap: 0;
  padding: 0;
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
  background: transparent;
  border: 0;
  border-bottom: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  border-radius: 0;

  &::-webkit-scrollbar {
    display: none;
  }
`;

export const TabBtn = styled.button<{ $active?: boolean }>`
  appearance: none;
  border: 0;
  cursor: pointer;
  font: inherit;
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  padding: 8px 12px 7px;
  border-radius: 0;
  color: ${({ theme, $active }) =>
    $active ? theme.semantic.text.primary : theme.semantic.text.secondary};
  background: transparent;
  border-bottom: 2px solid
    ${({ theme, $active }) => ($active ? theme.semantic.accent.default : 'transparent')};
  margin-bottom: -1px;
  box-shadow: none;
  transition:
    color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
    border-color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard};

  &:hover {
    color: ${({ theme }) => theme.semantic.text.primary};
  }
`;
