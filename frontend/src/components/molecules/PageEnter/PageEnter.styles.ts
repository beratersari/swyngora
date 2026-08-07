import styled, { keyframes } from 'styled-components';

const enter = keyframes`
  from {
    opacity: 0;
    transform: translate3d(0, 10px, 0);
  }
  to {
    opacity: 1;
    transform: translate3d(0, 0, 0);
  }
`;

export const Stage = styled.div`
  animation: ${enter} ${({ theme }) => theme.motion.duration.base} ${({ theme }) => theme.motion.ease.enter} both;

  @media (prefers-reduced-motion: reduce) {
    animation: none;
  }
`;
