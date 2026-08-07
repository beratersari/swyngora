import styled, { keyframes } from 'styled-components';

const shimmer = keyframes`
  0% {
    background-position: 100% 0;
  }
  100% {
    background-position: 0 0;
  }
`;

/** Stronger ant skeleton contrast on dark green surfaces */
export const SkeletonBlock = styled.div`
  display: block;

  .ant-skeleton .ant-skeleton-content .ant-skeleton-title,
  .ant-skeleton .ant-skeleton-content .ant-skeleton-paragraph > li {
    background: linear-gradient(
      90deg,
      ${({ theme }) => theme.semantic.skeleton.base} 25%,
      ${({ theme }) => theme.semantic.skeleton.highlight} 37%,
      ${({ theme }) => theme.semantic.skeleton.base} 63%
    ) !important;
    background-size: 400% 100% !important;
  }
`;

/** Chart / card / image block placeholder with high-contrast brand shimmer */
export const SkeletonChartBlock = styled.div`
  display: block;
  background: linear-gradient(
    90deg,
    ${({ theme }) => theme.semantic.skeleton.base} 0%,
    ${({ theme }) => theme.semantic.skeleton.mid} 35%,
    ${({ theme }) => theme.semantic.skeleton.highlight} 50%,
    ${({ theme }) => theme.semantic.skeleton.mid} 65%,
    ${({ theme }) => theme.semantic.skeleton.base} 100%
  );
  background-size: 220% 100%;
  animation: ${shimmer} 1.2s ease infinite;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
`;
