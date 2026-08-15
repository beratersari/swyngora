import styled, { css } from 'styled-components';
import { Tag } from 'antd';
import type { BrandTagVariant } from './BrandTag.types';

const variants: Record<BrandTagVariant, ReturnType<typeof css>> = {
  status: css`
    background: ${({ theme }) => theme.semantic.bg.chrome} !important;
    border: 1px solid ${({ theme }) => theme.semantic.border.default} !important;
    color: ${({ theme }) => theme.semantic.text.secondary} !important;
  `,
  live: css`
    background: ${({ theme }) => theme.semantic.bg.accentSoft} !important;
    border: 1px solid ${({ theme }) => theme.semantic.border.accent} !important;
    color: ${({ theme }) => theme.semantic.accent.default} !important;
  `,
  exchange: css`
    background: transparent !important;
    border: 1px solid ${({ theme }) => theme.semantic.border.default} !important;
    color: ${({ theme }) => theme.semantic.text.secondary} !important;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    font-size: 10px;
    font-weight: 700;
  `,
  paused: css`
    background: ${({ theme }) => theme.semantic.bg.chrome} !important;
    border: 1px solid ${({ theme }) => theme.semantic.border.subtle} !important;
    color: ${({ theme }) => theme.semantic.text.tertiary} !important;
  `,
  delist: css`
    background: rgba(224, 184, 106, 0.16) !important;
    border: 1px solid ${({ theme }) => theme.semantic.status.warning} !important;
    color: ${({ theme }) => theme.semantic.status.warning} !important;
  `,
  up: css`
    background: ${({ theme }) => theme.semantic.bg.successSoft} !important;
    border: 1px solid ${({ theme }) => theme.semantic.chart.up} !important;
    color: ${({ theme }) => theme.semantic.chart.up} !important;
    font-family: ${({ theme }) => theme.fontFamilies.mono};
    font-variant-numeric: tabular-nums;
  `,
  down: css`
    background: ${({ theme }) => theme.semantic.bg.dangerSoft} !important;
    border: 1px solid ${({ theme }) => theme.semantic.border.danger} !important;
    color: ${({ theme }) => theme.semantic.chart.down} !important;
    font-family: ${({ theme }) => theme.fontFamilies.mono};
    font-variant-numeric: tabular-nums;
  `,
  outline: css`
    background: transparent !important;
    border: 1px solid ${({ theme }) => theme.semantic.border.default} !important;
    color: ${({ theme }) => theme.semantic.text.secondary} !important;
  `,
  gradeA: css`
    background: ${({ theme }) => theme.semantic.bg.accentSoft} !important;
    border: 1px solid ${({ theme }) => theme.semantic.border.accent} !important;
    color: ${({ theme }) => theme.semantic.accent.default} !important;
    font-weight: 700;
  `,
  gradeB: css`
    background: rgba(224, 184, 106, 0.16) !important;
    border: 1px solid ${({ theme }) => theme.semantic.status.warning} !important;
    color: ${({ theme }) => theme.semantic.status.warning} !important;
    font-weight: 700;
  `,
  gradeC: css`
    background: ${({ theme }) => theme.semantic.bg.chrome} !important;
    border: 1px solid ${({ theme }) => theme.semantic.border.subtle} !important;
    color: ${({ theme }) => theme.semantic.text.tertiary} !important;
    font-weight: 700;
  `,
};

export const StyledBrandTag = styled(Tag)<{ $variant: BrandTagVariant }>`
  && {
    margin-inline-end: 0;
    line-height: 1.4;
    font-weight: 500;
    transition:
      background ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
      border-color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
      color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard};
    ${({ $variant }) => variants[$variant]}
  }
`;
