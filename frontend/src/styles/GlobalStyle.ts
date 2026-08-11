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
    transition: color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard};
  }

  a:hover {
    color: ${({ theme }) => theme.semantic.text.linkHover};
  }

  /* Desk chrome: snappy hovers, no layout thrash. */
  .ant-btn {
    transition:
      background ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
      border-color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
      color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
      transform ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
      box-shadow ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard} !important;
  }

  .ant-btn:not(:disabled):hover {
    transform: translateY(-1px);
  }

  .ant-btn:not(:disabled):active {
    transform: translateY(0);
  }

  .ant-input,
  .ant-input-affix-wrapper,
  .ant-select-selector,
  .ant-input-number,
  .ant-switch {
    transition:
      border-color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
      box-shadow ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
      background ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard} !important;
  }

  .ant-table-tbody > tr > td {
    transition: background-color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard};
  }

  .ant-tabs-ink-bar {
    transition: width ${({ theme }) => theme.motion.duration.base} ${({ theme }) => theme.motion.ease.enter},
      left ${({ theme }) => theme.motion.duration.base} ${({ theme }) => theme.motion.ease.enter} !important;
  }

  @media (prefers-reduced-motion: reduce) {
    html {
      scroll-behavior: auto;
    }

    *,
    *::before,
    *::after {
      animation-duration: 0.01ms !important;
      animation-iteration-count: 1 !important;
      transition-duration: 0.01ms !important;
      scroll-behavior: auto !important;
    }

    .ant-btn:not(:disabled):hover,
    .ant-btn:not(:disabled):active {
      transform: none;
    }
  }

  ::selection {
    background: ${({ theme }) => theme.semantic.bg.accentSoft};
    color: ${({ theme }) => theme.semantic.text.primary};
  }

  html {
    scrollbar-color: ${({ theme }) => theme.semantic.border.strong} ${({ theme }) => theme.semantic.bg.canvas};
  }

  *::-webkit-scrollbar {
    width: 10px;
    height: 10px;
  }

  *::-webkit-scrollbar-thumb {
    background: ${({ theme }) => theme.semantic.border.strong};
    border-radius: 999px;
  }

  *::-webkit-scrollbar-track {
    background: ${({ theme }) => theme.semantic.bg.canvas};
  }

  /* Ant Design surfaces that leak light defaults */
  .ant-input,
  .ant-input-affix-wrapper,
  .ant-select-selector,
  .ant-picker,
  .ant-input-number,
  .ant-input-number-input {
    background: ${({ theme }) => theme.semantic.bg.chrome} !important;
    color: ${({ theme }) => theme.semantic.text.primary} !important;
  }

  .ant-select-selection-item,
  .ant-select-selection-placeholder,
  .ant-input::placeholder,
  .ant-input-number-input::placeholder {
    color: ${({ theme }) => theme.semantic.text.secondary} !important;
  }

  .ant-select-selection-placeholder,
  .ant-input::placeholder {
    opacity: 1 !important;
  }

  .ant-select-dropdown,
  .ant-dropdown-menu,
  .ant-picker-dropdown,
  .ant-popover-inner,
  .ant-popconfirm .ant-popover-inner {
    background: ${({ theme }) => theme.semantic.bg.elevated} !important;
    color: ${({ theme }) => theme.semantic.text.primary} !important;
  }

  .ant-select-item,
  .ant-dropdown-menu-item,
  .ant-select-item-option-content {
    color: ${({ theme }) => theme.semantic.text.primary} !important;
  }

  .ant-select-item-option-selected {
    background: ${({ theme }) => theme.semantic.bg.accentSoft} !important;
    color: ${({ theme }) => theme.semantic.text.primary} !important;
  }

  .ant-empty-description {
    color: ${({ theme }) => theme.semantic.text.secondary} !important;
  }

  .ant-btn-link {
    color: ${({ theme }) => theme.semantic.text.link} !important;
  }

  .ant-btn-link:hover {
    color: ${({ theme }) => theme.semantic.text.linkHover} !important;
  }

  .ant-btn-text {
    color: ${({ theme }) => theme.semantic.text.primary} !important;
  }

  .ant-switch {
    background: ${({ theme }) => theme.palette.stone} !important;
  }

  .ant-switch-checked {
    background: ${({ theme }) => theme.palette.bangladeshGreen} !important;
  }

  /*
   * Ant Tag presets (processing/success/…) use light-mode hue math:
   * mid-green text on pale-green fill — unreadable on the desk.
   * Force every leftover Tag onto dark, high-contrast chips.
   */
  .ant-tag,
  .ant-tag-default,
  .ant-tag-processing,
  .ant-tag-success,
  .ant-tag-error,
  .ant-tag-warning,
  .ant-tag-blue,
  .ant-tag-cyan,
  .ant-tag-green,
  .ant-tag-geekblue,
  .ant-tag-has-color {
    background: ${({ theme }) => theme.palette.bangladeshGreen} !important;
    border-color: ${({ theme }) => theme.palette.frog} !important;
    color: ${({ theme }) => theme.palette.antiFlashWhite} !important;
  }
`;
