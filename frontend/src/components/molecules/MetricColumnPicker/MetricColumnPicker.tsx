import { useMemo, useState } from 'react';
import { Button, Checkbox, Popover } from 'antd';
import {
  CaretDownOutlined,
  CaretUpOutlined,
  ColumnHeightOutlined,
  HolderOutlined,
} from '@ant-design/icons';
import type { SpotMetricId } from '@/libs/utils/spotMetrics';
import { moveIdByDelta, moveItemAtIndex, toggleIdInList } from './MetricColumnPicker.helpers';
import {
  AvailableRow,
  AvailableSection,
  AvailableTitle,
  DragHandle,
  MetricLabel,
  MetricList,
  MetricRow,
  OrderButtons,
  Panel,
  PanelHint,
  PickerWrap,
} from './MetricColumnPicker.styles';
import type { MetricColumnPickerProps } from './MetricColumnPicker.types';

/**
 * Show/hide and reorder metric columns. Selection order = table column order.
 * Prefs are persisted by the parent (`useSpotMetricColumns`).
 */
export function MetricColumnPicker({
  available,
  value,
  onChange,
  onReset,
  getLabel,
  ariaLabel = 'Columns',
  resetLabel = 'Reset',
  buttonLabel = 'Columns',
  moveUpLabel = 'Move up',
  moveDownLabel = 'Move down',
  dragHintLabel = 'Drag rows or use arrows to change column order.',
}: MetricColumnPickerProps) {
  const [open, setOpen] = useState(false);
  const [dragId, setDragId] = useState<SpotMetricId | null>(null);
  const [overId, setOverId] = useState<SpotMetricId | null>(null);

  const byId = useMemo(() => {
    const map = new Map(available.map((m) => [m.id, m]));
    return map;
  }, [available]);

  const selectedDefs = value.map((id) => byId.get(id)).filter(Boolean);
  const unselected = available.filter((m) => !value.includes(m.id));

  const commit = (next: SpotMetricId[]) => {
    if (next.length === 0 && available[0]) {
      onChange([available[0].id]);
      return;
    }
    onChange(next);
  };

  const panel = (
    <Panel>
      <PanelHint>{dragHintLabel}</PanelHint>
      <MetricList aria-label={ariaLabel}>
        {selectedDefs.map((def, index) => {
          if (!def) return null;
          const id = def.id;
          return (
            <MetricRow
              key={id}
              $dragging={dragId === id}
              $dragOver={overId === id && dragId !== id}
              draggable
              onDragStart={(e) => {
                setDragId(id);
                e.dataTransfer.effectAllowed = 'move';
                e.dataTransfer.setData('text/plain', id);
              }}
              onDragEnd={() => {
                setDragId(null);
                setOverId(null);
              }}
              onDragOver={(e) => {
                e.preventDefault();
                e.dataTransfer.dropEffect = 'move';
                if (overId !== id) setOverId(id);
              }}
              onDragLeave={() => {
                if (overId === id) setOverId(null);
              }}
              onDrop={(e) => {
                e.preventDefault();
                const fromId = (e.dataTransfer.getData('text/plain') || dragId) as SpotMetricId;
                const from = value.indexOf(fromId);
                const to = value.indexOf(id);
                if (from >= 0 && to >= 0 && from !== to) {
                  commit(moveItemAtIndex(value, from, to));
                }
                setDragId(null);
                setOverId(null);
              }}
            >
              <DragHandle aria-hidden title={dragHintLabel}>
                <HolderOutlined />
              </DragHandle>
              <MetricLabel>{getLabel(def.labelKey)}</MetricLabel>
              <OrderButtons>
                <Button
                  type="text"
                  size="small"
                  icon={<CaretUpOutlined />}
                  disabled={index === 0}
                  aria-label={`${moveUpLabel}: ${getLabel(def.labelKey)}`}
                  onClick={() => commit(moveIdByDelta(value, id, -1))}
                />
                <Button
                  type="text"
                  size="small"
                  icon={<CaretDownOutlined />}
                  disabled={index === value.length - 1}
                  aria-label={`${moveDownLabel}: ${getLabel(def.labelKey)}`}
                  onClick={() => commit(moveIdByDelta(value, id, 1))}
                />
              </OrderButtons>
              <Checkbox
                checked
                disabled={value.length <= 1}
                aria-label={getLabel(def.labelKey)}
                onChange={() => commit(toggleIdInList(value, id))}
              />
            </MetricRow>
          );
        })}
      </MetricList>

      {unselected.length > 0 ? (
        <AvailableSection>
          <AvailableTitle>{buttonLabel}</AvailableTitle>
          {unselected.map((def) => (
            <AvailableRow key={def.id}>
              <Checkbox
                checked={false}
                aria-label={getLabel(def.labelKey)}
                onChange={() => commit(toggleIdInList(value, def.id))}
              />
              <span>{getLabel(def.labelKey)}</span>
            </AvailableRow>
          ))}
        </AvailableSection>
      ) : null}
    </Panel>
  );

  return (
    <PickerWrap>
      <Popover
        trigger="click"
        open={open}
        onOpenChange={setOpen}
        placement="bottomLeft"
        content={panel}
      >
        <Button icon={<ColumnHeightOutlined />} aria-label={ariaLabel}>
          {buttonLabel}
        </Button>
      </Popover>
      {onReset ? (
        <Button type="link" size="small" onClick={onReset}>
          {resetLabel}
        </Button>
      ) : null}
    </PickerWrap>
  );
}
