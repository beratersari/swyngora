import styled from 'styled-components';

export const Rail = styled.div`
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px;
  background: ${({ theme }) => theme.semantic.bg.chrome};
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  border-radius: ${({ theme }) => theme.radii.pill}px;
  overflow-x: auto;
  max-width: 100%;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;

  &::-webkit-scrollbar {
    display: none;
  }
`;

export const Chip = styled.button<{ $active?: boolean }>`
  appearance: none;
  border: 0;
  cursor: pointer;
  font: inherit;
  font-size: 12px;
  font-weight: 600;
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  letter-spacing: 0.02em;
  padding: 4px 10px;
  border-radius: ${({ theme }) => theme.radii.pill}px;
  color: ${({ theme, $active }) =>
    $active ? theme.semantic.text.primary : theme.semantic.text.secondary};
  background: ${({ theme, $active }) =>
    $active ? theme.semantic.bg.accentSoft : 'transparent'};
  box-shadow: ${({ theme, $active }) =>
    $active ? `inset 0 0 0 1px ${theme.semantic.border.accent}` : 'none'};
  transition:
    color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
    background ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard};

  &:hover {
    color: ${({ theme }) => theme.semantic.text.primary};
  }
`;
