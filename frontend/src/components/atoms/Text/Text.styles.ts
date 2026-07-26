import styled, { css } from 'styled-components';

export type TextRootProps = {
  $truncate?: boolean;
  $mono?: boolean;
};

export const TextRoot = styled.span<TextRootProps>`
  font-family: ${({ theme }) => theme.fontFamilies.sans};

  ${({ $truncate }) =>
    $truncate &&
    css`
      display: block;
      max-width: 100%;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    `}

  ${({ $mono, theme }) =>
    $mono &&
    css`
      font-family: ${theme.fontFamilies.mono};
    `}
`;
