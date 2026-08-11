import styled from 'styled-components';

export const Wrap = styled.header`
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const Copy = styled.div`
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  max-width: 72ch;
`;

export const Eyebrow = styled.p`
  margin: 0;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: ${({ theme }) => theme.semantic.text.accent};
`;

export const Extra = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;

  ${({ theme }) => theme.media.phone} {
    width: 100%;
  }
`;
