import styled, { css, keyframes } from 'styled-components';

const flashUp = keyframes`
  from { background-color: rgba(22, 199, 132, 0.22); }
  to { background-color: transparent; }
`;

const flashDown = keyframes`
  from { background-color: rgba(234, 57, 67, 0.2); }
  to { background-color: transparent; }
`;

export const FlashWrap = styled.span<{ $dir: 'up' | 'down' | null }>`
  display: inline-block;
  border-radius: 4px;
  padding: 0 3px;
  margin: 0 -3px;
  ${({ $dir, theme }) =>
    $dir === 'up'
      ? css`
          animation: ${flashUp} ${theme.motion.duration.slow} ${theme.motion.ease.exit};
        `
      : $dir === 'down'
        ? css`
            animation: ${flashDown} ${theme.motion.duration.slow} ${theme.motion.ease.exit};
          `
        : null};

  @media (prefers-reduced-motion: reduce) {
    animation: none;
  }
`;
