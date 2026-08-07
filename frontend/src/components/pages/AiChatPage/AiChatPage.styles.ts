import styled from 'styled-components';

/**
 * AI chat surfaces use solid palette colors (not translucent washes).
 * Goal: ≥7:1 body contrast — anti-flash white on rich black / forest.
 */

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[4]}px;
  min-height: min(70vh, 720px);
`;

export const PageIntro = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[1]}px;
`;

export const Thread = styled.div`
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-height: 280px;
  max-height: min(55vh, 520px);
  overflow-y: auto;
  padding: ${({ theme }) => theme.spacing[3]}px;
  border-radius: ${({ theme }) => theme.radii.lg}px;
  border: 1px solid ${({ theme }) => theme.palette.bangladeshGreen};
  background: ${({ theme }) => theme.palette.richBlack};
`;

export const BubbleRow = styled.div<{ $role: 'user' | 'assistant' | 'system' }>`
  display: flex;
  justify-content: ${({ $role }) => ($role === 'user' ? 'flex-end' : 'flex-start')};
`;

export const Bubble = styled.div<{ $role: 'user' | 'assistant' | 'system'; $error?: boolean }>`
  max-width: min(720px, 92%);
  padding: ${({ theme }) => `${theme.spacing[3]}px ${theme.spacing[4]}px`};
  border-radius: ${({ theme }) => theme.radii.md}px;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.5;
  border: 1px solid
    ${({ theme, $error, $role }) =>
      $error
        ? '#E07A7A'
        : $role === 'user'
          ? theme.palette.mountainMeadow
          : theme.palette.bangladeshGreen};
  background: ${({ theme, $role, $error }) =>
    $error
      ? '#2A1212'
      : $role === 'user'
        ? theme.palette.forest
        : theme.palette.darkGreen};
  color: ${({ theme }) => theme.palette.antiFlashWhite};

  /* Message body only — do not override MetaChip colors */
  & [data-text-role='body'] {
    color: ${({ theme }) => theme.palette.antiFlashWhite} !important;
  }
`;

export const MetaRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
  margin-top: ${({ theme }) => theme.spacing[3]}px;
  padding-top: ${({ theme }) => theme.spacing[2]}px;
  border-top: 1px solid ${({ theme }) => theme.palette.bangladeshGreen};
`;

export const Composer = styled.form`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const ComposerRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: flex-end;
`;

export const Suggestions = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const EmptyState = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  align-items: flex-start;
  padding: ${({ theme }) => theme.spacing[2]}px 0;
`;

export const ToolbarRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: center;
`;

/**
 * Info / disclaimer panel — solid rich black + anti-flash white (not translucent green).
 * Previous chrome + soft accent looked like “same green family” and was hard to read.
 */
export const DisclaimerBanner = styled.aside`
  display: flex;
  align-items: flex-start;
  gap: ${({ theme }) => theme.spacing[3]}px;
  padding: ${({ theme }) => theme.spacing[3]}px ${({ theme }) => theme.spacing[4]}px;
  border-radius: ${({ theme }) => theme.radii.md}px;
  border: 1px solid ${({ theme }) => theme.palette.mountainMeadow};
  background: ${({ theme }) => theme.palette.richBlack};
  color: ${({ theme }) => theme.palette.antiFlashWhite};
  box-shadow: inset 3px 0 0 ${({ theme }) => theme.palette.mountainMeadow};
`;

export const DisclaimerIcon = styled.span`
  flex-shrink: 0;
  width: 10px;
  height: 10px;
  margin-top: 5px;
  border-radius: 50%;
  background: ${({ theme }) => theme.palette.mountainMeadow};
`;

export const DisclaimerBody = styled.div`
  flex: 1;
  min-width: 0;
  color: ${({ theme }) => theme.palette.antiFlashWhite} !important;
  font-size: 14px;
  line-height: 1.5;
  font-weight: 500;
`;

/**
 * Tool chips: solid basil fill + white text + meadow border (readable on dark bubbles).
 * Thinking chips: solid pine + pistachio text.
 */
export const MetaChip = styled.span<{ $kind?: 'tool' | 'thinking' }>`
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  padding: 4px 12px;
  border-radius: ${({ theme }) => theme.radii.pill}px;
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 12px;
  font-weight: 600;
  line-height: 1.45;
  word-break: break-all;
  color: ${({ theme, $kind }) =>
    $kind === 'thinking' ? theme.palette.pistachio : theme.palette.antiFlashWhite};
  background: ${({ theme, $kind }) =>
    $kind === 'thinking' ? theme.palette.pine : theme.palette.basil};
  border: 1px solid
    ${({ theme, $kind }) =>
      $kind === 'thinking' ? theme.palette.frog : theme.palette.mountainMeadow};
`;

export const MetaLabel = styled.span`
  display: block;
  width: 100%;
  margin-bottom: 4px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: ${({ theme }) => theme.palette.pistachio};
`;
