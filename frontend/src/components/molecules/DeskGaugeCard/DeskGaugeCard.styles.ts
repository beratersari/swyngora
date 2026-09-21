import styled from 'styled-components';

export const Card = styled.button`
  display: flex;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
  width: 100%;
  min-width: 0;
  min-height: 112px;
  margin: 0;
  padding: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[3]}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  border-radius: ${({ theme }) => theme.radii.md}px;
  background: ${({ theme }) => theme.semantic.bg.muted};
  color: inherit;
  cursor: pointer;
  text-align: left;

  &:hover {
    border-color: ${({ theme }) => theme.semantic.border.strong};
  }
`;

export const Copy = styled.span`
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
`;
