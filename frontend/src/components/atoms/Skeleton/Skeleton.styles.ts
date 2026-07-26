import styled, { keyframes } from 'styled-components';

const shimmer = keyframes`
  0% {
    background-position: 100% 0;
  }
  100% {
    background-position: 0 0;
  }
`;

export const SkeletonBlock = styled.div`
  display: block;
`;

/** Chart / card / image block placeholder with brand shimmer */
export const SkeletonChartBlock = styled.div`
  display: block;
  background: linear-gradient(
    90deg,
    ${({ theme }) => theme.semantic.skeleton.base} 25%,
    ${({ theme }) => theme.semantic.skeleton.highlight} 37%,
    ${({ theme }) => theme.semantic.skeleton.base} 63%
  );
  background-size: 400% 100%;
  animation: ${shimmer} 1.4s ease infinite;
`;
