import styled, { css, keyframes } from 'styled-components';
import { Button } from 'antd';
import { motion } from '@/styles/tokens';

const pop = keyframes`
  0% { transform: scale(0.55) rotate(-18deg); }
  55% { transform: scale(1.22) rotate(10deg); }
  100% { transform: scale(1) rotate(0deg); }
`;

export const StarButton = styled(Button)<{ $watched: boolean }>`
  && {
    color: ${({ theme, $watched }) =>
      $watched ? theme.semantic.accent.default : theme.semantic.text.secondary};
    transition:
      color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
      transform ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard};

    .anticon {
      font-size: 16px;
      ${({ $watched }) =>
        $watched
          ? css`
              animation: ${pop} ${motion.duration.base} ${motion.ease.enter};
            `
          : null};
    }

    &:hover {
      color: ${({ theme }) => theme.semantic.accent.default} !important;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    && .anticon {
      animation: none;
    }
  }
`;
