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
  max-height: min(62vh, 640px);
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
  word-break: break-word;
  line-height: 1.55;
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

export const UserBody = styled.div`
  white-space: pre-wrap;
  color: ${({ theme }) => theme.palette.antiFlashWhite};
`;

export const MdStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: 0.65em;
  color: ${({ theme }) => theme.palette.antiFlashWhite};
`;

export const MdP = styled.p`
  margin: 0;
`;

export const MdList = styled.ul`
  margin: 0;
  padding-left: 1.2em;
`;

export const MdOl = styled.ol`
  margin: 0;
  padding-left: 1.2em;
`;

export const MdPre = styled.pre`
  margin: 0;
  padding: 8px 10px;
  overflow-x: auto;
  border-radius: ${({ theme }) => theme.radii.sm}px;
  background: ${({ theme }) => theme.palette.richBlack};
  border: 1px solid ${({ theme }) => theme.palette.bangladeshGreen};
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 12px;
  line-height: 1.45;
  white-space: pre-wrap;
`;

export const MdStrong = styled.strong`
  font-weight: 700;
  color: ${({ theme }) => theme.palette.antiFlashWhite};
`;

export const MdCode = styled.code`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 0.92em;
  padding: 0 4px;
  border-radius: 4px;
  background: ${({ theme }) => theme.palette.pine};
`;

export const TraceDetails = styled.details`
  margin-top: ${({ theme }) => theme.spacing[3]}px;
  padding-top: ${({ theme }) => theme.spacing[2]}px;
  border-top: 1px solid ${({ theme }) => theme.palette.bangladeshGreen};

  summary {
    cursor: pointer;
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: ${({ theme }) => theme.palette.pistachio};
    list-style: none;
  }

  summary::-webkit-details-marker {
    display: none;
  }

  summary::before {
    content: '▸ ';
    color: ${({ theme }) => theme.palette.mountainMeadow};
  }

  &[open] summary::before {
    content: '▾ ';
  }
`;

export const TraceList = styled.ol`
  margin: ${({ theme }) => theme.spacing[2]}px 0 0;
  padding-left: 1.2em;
  color: ${({ theme }) => theme.palette.pistachio};
  font-size: 12px;
  line-height: 1.45;
`;

export const ProcessPanel = styled.details`
  margin: ${({ theme }) => theme.spacing[2]}px 0 ${({ theme }) => theme.spacing[3]}px;
  padding: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[3]}px;
  border-radius: ${({ theme }) => theme.radii.sm}px;
  border: 1px dashed ${({ theme }) => theme.palette.frog};
  background: ${({ theme }) => theme.palette.richBlack};
`;

export const ProcessTitle = styled.summary`
  cursor: pointer;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: ${({ theme }) => theme.palette.pistachio};
  user-select: none;

  &::-webkit-details-marker {
    display: none;
  }

  &::before {
    content: '▸ ';
    color: ${({ theme }) => theme.palette.mountainMeadow};
  }

  ${ProcessPanel}[open] &::before {
    content: '▾ ';
  }

  ${ProcessPanel}[open] & {
    margin-bottom: ${({ theme }) => theme.spacing[2]}px;
  }
`;

export const ProcessPreview = styled.span`
  display: block;
  max-width: 100%;
  padding-left: 14px;
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0;
  text-transform: none;
  color: ${({ theme }) => theme.palette.antiFlashWhite};
  opacity: 0.82;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
`;

export const ProcessList = styled.ol`
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 220px;
  overflow-y: auto;
`;

export const ProcessItem = styled.li<{ $kind: string; $active?: boolean }>`
  display: grid;
  grid-template-columns: 22px 64px 1fr;
  gap: 8px;
  align-items: start;
  font-size: 12px;
  line-height: 1.45;
  color: ${({ theme }) => theme.palette.antiFlashWhite};
  opacity: ${({ $active }) => ($active ? 1 : 0.88)};
  padding-left: 6px;
  border-left: 2px solid
    ${({ theme, $active }) => ($active ? theme.palette.mountainMeadow : 'transparent')};
`;

export const ProcessIndex = styled.span`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 11px;
  color: ${({ theme }) => theme.palette.mountainMeadow};
  padding-top: 1px;
`;

export const ProcessKind = styled.span<{ $kind: string }>`
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  padding-top: 2px;
  color: ${({ theme, $kind }) =>
    $kind === 'tool' || $kind === 'tool_result'
      ? theme.palette.mint
      : $kind === 'tool_error'
        ? '#E07A7A'
        : theme.palette.pistachio};
`;

export const ProcessText = styled.span`
  min-width: 0;
  color: ${({ theme }) => theme.palette.antiFlashWhite};
  word-break: break-word;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
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
  max-width: 16rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 3px 10px;
  border-radius: ${({ theme }) => theme.radii.pill}px;
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 12px;
  font-weight: 600;
  line-height: 1.45;
  color: ${({ theme, $kind }) =>
    $kind === 'thinking' ? theme.palette.pistachio : theme.palette.antiFlashWhite};
  background: ${({ theme, $kind }) =>
    $kind === 'thinking' ? theme.palette.pine : theme.palette.basil};
  border: 1px solid
    ${({ theme, $kind }) =>
      $kind === 'thinking' ? theme.palette.frog : theme.palette.mountainMeadow};
`;

export const RefList = styled.ol`
  margin: ${({ theme }) => theme.spacing[2]}px 0 0;
  padding: 0;
  width: 100%;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
`;

export const RefItem = styled.li`
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 10px;
  border-radius: ${({ theme }) => theme.radii.sm}px;
  border: 1px solid ${({ theme }) => theme.palette.bangladeshGreen};
  background: ${({ theme }) => theme.palette.richBlack};
`;

export const RefLink = styled.a`
  color: ${({ theme }) => theme.palette.mint} !important;
  font-weight: 600;
  text-decoration: none;
  word-break: break-word;

  &:hover {
    color: ${({ theme }) => theme.palette.caribbeanGreen} !important;
    text-decoration: underline;
  }
`;

export const RefUrl = styled.span`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 11px;
  color: ${({ theme }) => theme.palette.pistachio};
  word-break: break-all;
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
