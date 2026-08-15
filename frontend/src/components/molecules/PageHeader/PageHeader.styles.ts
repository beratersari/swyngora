import styled from 'styled-components';

export const Wrap = styled.header`
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-height: 28px;
  margin-bottom: ${({ theme }) => theme.spacing[2]}px;
`;

export const Copy = styled.div`
  display: flex;
  flex-direction: row;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 6px ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;

  h1 {
    margin: 0;
    font-size: 13px !important;
    line-height: 1.2 !important;
    letter-spacing: 0.1em;
    text-transform: uppercase;
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
