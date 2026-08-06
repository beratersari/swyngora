import { createGlobalStyle } from 'styled-components';

/**
 * Global styles via styled-components — no standalone CSS files.
 * Fonts: Google Fonts with system fallbacks from theme.fontFamilies.
 */
export const GlobalStyle = createGlobalStyle`
  @import url('https://fonts.googleapis.com/css2?family=DM+Sans:ital,opsz,wght@0,9..40,400;0,9..40,500;0,9..40,600;0,9..40,700;1,9..40,400&family=JetBrains+Mono:wght@400;500;600&display=swap');

  *,
  *::before,
  *::after {
    box-sizing: border-box;
  }

  html,
  body,
  #root {
    height: 100%;
    margin: 0;
  }

  body {
    font-family: ${({ theme }) => theme.fontFamilies.sans};
    font-size: ${({ theme }) => theme.typeScale.body.fontSize}px;
    line-height: ${({ theme }) => theme.typeScale.body.lineHeight};
    background: ${({ theme }) => theme.semantic.bg.page};
    color: ${({ theme }) => theme.semantic.text.primary};
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
  }

  code,
  pre {
    font-family: ${({ theme }) => theme.fontFamilies.mono};
  }

  /* Focus ring: caribbean green reserved for accessibility focus, not decoration */
  :focus-visible {
    outline: 2px solid ${({ theme }) => theme.semantic.border.focus};
    outline-offset: 2px;
  }

  a {
    color: ${({ theme }) => theme.semantic.text.link};
  }

  a:hover {
    color: ${({ theme }) => theme.semantic.text.linkHover};
  }

  /* Ant Design surfaces that leak light defaults */
  .ant-input,
  .ant-input-affix-wrapper,
  .ant-select-selector,
  .ant-picker {
    background: ${({ theme }) => theme.semantic.bg.chrome} !important;
    color: ${({ theme }) => theme.semantic.text.primary} !important;
  }

  .ant-select-dropdown,
  .ant-dropdown-menu,
  .ant-picker-dropdown {
    background: ${({ theme }) => theme.semantic.bg.elevated} !important;
  }
`;
