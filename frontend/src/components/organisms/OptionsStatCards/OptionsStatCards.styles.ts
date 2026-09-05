import styled from 'styled-components';

export const Grid = styled.div`
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: ${({ theme }) => theme.spacing[2]}px;

  @media (max-width: 1100px) {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  @media (max-width: 720px) {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
`;

export const Card = styled.article`
  min-width: 0;
  padding: ${({ theme }) => theme.spacing[3]}px;
  border-radius: ${({ theme }) => theme.radii.md}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  background: ${({ theme }) => theme.semantic.bg.elevated};
`;

export const Label = styled.div`
  margin-bottom: 6px;
`;
