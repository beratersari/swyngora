import styled from 'styled-components';

export const Wrap = styled.header`
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[3]}px;
  margin-bottom: ${({ theme }) => theme.spacing[4]}px;
`;

export const Copy = styled.div`
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;

  h1 {
    margin: 0;
    font-size: 28px !important;
    line-height: 1.2 !important;
    letter-spacing: -0.03em;
    text-transform: none;
    font-weight: 800 !important;
  }
`;

export const Eyebrow = styled.p`
  margin: 0;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: ${({ theme }) => theme.semantic.text.tertiary};
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
