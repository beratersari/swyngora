import styled from 'styled-components';

export const PickerWrap = styled.div`
  display: inline-flex;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
  flex: 0 0 auto;
`;

export const Panel = styled.div`
  width: min(320px, 90vw);
  max-height: min(420px, 70vh);
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const PanelHint = styled.p`
  margin: 0;
  font-size: 12px;
  color: ${({ theme }) => theme.semantic.text.secondary};
  line-height: 1.4;
`;

export const MetricList = styled.ul`
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

export const MetricRow = styled.li<{ $dragging?: boolean; $dragOver?: boolean }>`
  display: grid;
  grid-template-columns: auto 1fr auto auto;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border-radius: ${({ theme }) => theme.radii.sm}px;
  border: 1px solid
    ${({ theme, $dragOver }) =>
      $dragOver ? theme.colors.caribbeanGreen : theme.semantic.border.default};
  background: ${({ theme, $dragging }) =>
    $dragging ? 'rgba(23, 135, 109, 0.35)' : theme.semantic.bg.muted};
  opacity: ${({ $dragging }) => ($dragging ? 0.7 : 1)};
  cursor: grab;

  &:active {
    cursor: grabbing;
  }
`;

export const DragHandle = styled.span`
  display: inline-flex;
  align-items: center;
  color: ${({ theme }) => theme.semantic.text.secondary};
  font-size: 12px;
  user-select: none;
  letter-spacing: -1px;
`;

export const MetricLabel = styled.span`
  font-size: 13px;
  color: ${({ theme }) => theme.semantic.text.primary};
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
`;

export const OrderButtons = styled.div`
  display: inline-flex;
  gap: 2px;
`;

export const AvailableSection = styled.div`
  border-top: 1px solid ${({ theme }) => theme.semantic.border.default};
  padding-top: ${({ theme }) => theme.spacing[2]}px;
  display: flex;
  flex-direction: column;
  gap: 6px;
`;

export const AvailableTitle = styled.div`
  font-size: 12px;
  font-weight: 600;
  color: ${({ theme }) => theme.semantic.text.secondary};
  text-transform: uppercase;
  letter-spacing: 0.04em;
`;

export const AvailableRow = styled.label`
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 2px;
  cursor: pointer;
  font-size: 13px;
  color: ${({ theme }) => theme.semantic.text.primary};
`;
