import styled from 'styled-components';

export const Strip = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
  max-width: 100%;

  ${({ theme }) => theme.media.tablet} {
    grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
    gap: ${({ theme }) => theme.spacing[2]}px;
  }

  ${({ theme }) => theme.media.phone} {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: ${({ theme }) => theme.spacing[2]}px;
  }

  ${({ theme }) => theme.media.xs} {
    grid-template-columns: 1fr;
  }
`;

export const Card = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: ${({ theme }) => theme.spacing[3]}px;
  border-radius: ${({ theme }) => theme.radii.md}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  background: ${({ theme }) => theme.semantic.bg.elevated};
  min-width: 0;
  overflow: hidden;

  ${({ theme }) => theme.media.phone} {
    padding: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[3]}px;
  }

  /* Long mono prices wrap instead of blowing the grid */
  > *:last-child {
    word-break: break-word;
    overflow-wrap: anywhere;
  }
`;
