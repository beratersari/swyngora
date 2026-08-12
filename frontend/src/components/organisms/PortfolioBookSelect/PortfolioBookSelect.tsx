import { useState } from 'react';
import { Button, Input, InputNumber, Modal, Select } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { Field, Row } from './PortfolioBookSelect.styles';
import type { PortfolioBookSelectProps } from './PortfolioBookSelect.types';

export function PortfolioBookSelect({
  books,
  selectedId,
  loading,
  creating,
  onSelect,
  onCreate,
}: PortfolioBookSelectProps) {
  const { t } = useTranslation(['portfolio', 'common']);
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('Main');
  const [balance, setBalance] = useState<number | null>(10000);

  const options = books.map((b) => ({
    value: b.id ?? '',
    label: `${b.name || b.id} · ${b.currency ?? 'USDT'}`,
  })).filter((o) => o.value);

  return (
    <Row>
      <Field>
        <Text variant="caption" color="secondary">
          {t('portfolio:books.label')}
        </Text>
        <Select
          loading={loading}
          placeholder={t('portfolio:books.select')}
          value={selectedId || undefined}
          options={options}
          onChange={onSelect}
          aria-label={t('portfolio:books.label')}
        />
      </Field>
      <Button type="default" onClick={() => setOpen(true)}>
        {t('portfolio:books.create')}
      </Button>
      <Modal
        title={t('portfolio:books.createTitle')}
        open={open}
        okText={t('portfolio:books.create')}
        confirmLoading={creating}
        onCancel={() => setOpen(false)}
        onOk={async () => {
          if (balance == null || balance <= 0) return;
          await onCreate({ name: name.trim() || 'Main', startingBalance: balance });
          setOpen(false);
        }}
      >
        <Field>
          <Text variant="caption" color="secondary">
            {t('portfolio:books.name')}
          </Text>
          <Input value={name} onChange={(e) => setName(e.target.value)} />
        </Field>
        <Field style={{ marginTop: 12 }}>
          <Text variant="caption" color="secondary">
            {t('portfolio:books.startingBalance')}
          </Text>
          <InputNumber
            min={1}
            style={{ width: '100%' }}
            value={balance}
            onChange={(v) => setBalance(typeof v === 'number' ? v : null)}
          />
        </Field>
      </Modal>
    </Row>
  );
}
