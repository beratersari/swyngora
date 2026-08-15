import styled from 'styled-components';

/**
 * AI chat — light consumer palette (same as the rest of the site).
 * Surfaces: paper / mist. Text: ink. Accent: brand blue.
 */

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[4]}px;
  min-height: min(70vh, 720px);
  min-width: 0;

  ${({ theme }) => theme.media.phone} {
    min-height: 0;
    gap: ${({ theme }) => theme.spacing[3]}px;
  }
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

  ${({ theme }) => theme.media.phone} {
    min-height: 42vh;
    max-height: min(58vh, 520px);
    padding: ${({ theme }) => theme.spacing[2]}px;
  }
  overflow-y: auto;
  padding: ${({ theme }) => theme.spacing[3]}px;
  border-radius: ${({ theme }) => theme.radii.lg}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  background: ${({ theme }) => theme.semantic.bg.canvas};
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
        ? theme.semantic.border.danger
        : $role === 'user'
          ? theme.semantic.border.accent
          : theme.semantic.border.default};
  background: ${({ theme, $role, $error }) =>
    $error
      ? theme.semantic.bg.dangerSoft
      : $role === 'user'
        ? theme.semantic.bg.accentSoft
        : theme.semantic.bg.page};
  color: ${({ theme }) => theme.semantic.text.primary};

  & [data-text-role='body'] {
    color: ${({ theme }) => theme.semantic.text.primary} !important;
  }
`;

export const UserBody = styled.div`
  white-space: pre-wrap;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const MdStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: 0.65em;
  color: ${({ theme }) => theme.semantic.text.primary};
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
  background: ${({ theme }) => theme.semantic.bg.page};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 12px;
  line-height: 1.45;
  white-space: pre-wrap;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const MdStrong = styled.strong`
  font-weight: 700;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const MdCode = styled.code`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 0.92em;
  padding: 0 4px;
  border-radius: 4px;
  background: ${({ theme }) => theme.semantic.bg.accentMuted};
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const TraceDetails = styled.details`
  margin-top: ${({ theme }) => theme.spacing[3]}px;
  padding-top: ${({ theme }) => theme.spacing[2]}px;
  border-top: 1px solid ${({ theme }) => theme.semantic.border.default};

  summary {
    cursor: pointer;
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: ${({ theme }) => theme.semantic.text.secondary};
    list-style: none;
  }

  summary::-webkit-details-marker {
    display: none;
  }

  summary::before {
    content: '▸ ';
    color: ${({ theme }) => theme.semantic.accent.default};
  }

  &[open] summary::before {
    content: '▾ ';
  }
`;

export const TraceList = styled.ol`
  margin: ${({ theme }) => theme.spacing[2]}px 0 0;
  padding-left: 1.2em;
  color: ${({ theme }) => theme.semantic.text.secondary};
  font-size: 12px;
  line-height: 1.45;
`;

export const ProcessPanel = styled.details`
  margin: ${({ theme }) => theme.spacing[2]}px 0 ${({ theme }) => theme.spacing[3]}px;
  padding: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[3]}px;
  border-radius: ${({ theme }) => theme.radii.sm}px;
  border: 1px dashed ${({ theme }) => theme.semantic.border.strong};
  background: ${({ theme }) => theme.semantic.bg.page};
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
  color: ${({ theme }) => theme.semantic.text.secondary};
  user-select: none;

  &::-webkit-details-marker {
    display: none;
  }

  &::before {
    content: '▸ ';
    color: ${({ theme }) => theme.semantic.accent.default};
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
  color: ${({ theme }) => theme.semantic.text.primary};
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

  ${({ theme }) => theme.media.phone} {
    grid-template-columns: 22px 1fr;
  }
  align-items: start;
  font-size: 12px;
  line-height: 1.45;
  color: ${({ theme }) => theme.semantic.text.primary};
  opacity: ${({ $active }) => ($active ? 1 : 0.88)};
  padding-left: 6px;
  border-left: 2px solid
    ${({ theme, $active }) => ($active ? theme.semantic.accent.default : 'transparent')};
`;

export const ProcessIndex = styled.span`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 11px;
  color: ${({ theme }) => theme.semantic.accent.default};
  padding-top: 1px;
`;

export const ProcessKind = styled.span<{ $kind: string }>`
  ${({ theme }) => theme.media.phone} {
    display: none;
  }

  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  padding-top: 2px;
  color: ${({ theme, $kind }) =>
    $kind === 'tool' || $kind === 'tool_result'
      ? theme.semantic.accent.default
      : $kind === 'tool_error'
        ? theme.semantic.status.error
        : theme.semantic.text.secondary};
`;

export const ProcessText = styled.span`
  min-width: 0;
  color: ${({ theme }) => theme.semantic.text.primary};
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
  border-top: 1px solid ${({ theme }) => theme.semantic.border.default};
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

  ${({ theme }) => theme.media.phone} {
    flex-direction: column;
    align-items: stretch;

    .ant-btn {
      width: 100%;
    }
  }
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

export const DisclaimerBanner = styled.aside`
  display: flex;
  align-items: flex-start;
  gap: ${({ theme }) => theme.spacing[3]}px;
  padding: ${({ theme }) => theme.spacing[3]}px ${({ theme }) => theme.spacing[4]}px;
  border-radius: ${({ theme }) => theme.radii.md}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  background: ${({ theme }) => theme.semantic.bg.page};
  color: ${({ theme }) => theme.semantic.text.primary};
  box-shadow: inset 3px 0 0 ${({ theme }) => theme.semantic.accent.default};
`;

export const DisclaimerIcon = styled.span`
  flex-shrink: 0;
  width: 10px;
  height: 10px;
  margin-top: 5px;
  border-radius: 50%;
  background: ${({ theme }) => theme.semantic.accent.default};
`;

export const DisclaimerBody = styled.div`
  flex: 1;
  min-width: 0;
  color: ${({ theme }) => theme.semantic.text.primary} !important;
  font-size: 14px;
  line-height: 1.5;
  font-weight: 500;
`;

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
  color: ${({ theme }) => theme.semantic.text.primary};
  background: ${({ theme, $kind }) =>
    $kind === 'thinking' ? theme.semantic.bg.page : theme.semantic.bg.accentSoft};
  border: 1px solid
    ${({ theme, $kind }) =>
      $kind === 'thinking' ? theme.semantic.border.default : theme.semantic.border.accent};
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
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  background: ${({ theme }) => theme.semantic.bg.page};
`;

export const RefLink = styled.a`
  color: ${({ theme }) => theme.semantic.text.link} !important;
  font-weight: 600;
  text-decoration: none;
  word-break: break-word;

  &:hover {
    color: ${({ theme }) => theme.semantic.text.linkHover} !important;
    text-decoration: underline;
  }
`;

export const RefUrl = styled.span`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 11px;
  color: ${({ theme }) => theme.semantic.text.tertiary};
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
  color: ${({ theme }) => theme.semantic.text.tertiary};
`;
