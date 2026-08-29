import { useState } from 'react';
import { Alert, Button, Form, Input, InputNumber, Select, Table, Tabs, Typography, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import {
  rtkErrorMessage,
  useCancelExportMutation,
  useCreateAccountAPIKeyMutation,
  useCreateRecurringBuyMutation,
  useDeleteRecurringBuyMutation,
  useListAccountAPIKeysQuery,
  useListExportsQuery,
  useListPortfolioSharesQuery,
  useListRecurringBuysQuery,
  useListWatchlistSharesQuery,
  usePauseRecurringBuyMutation,
  useResumeRecurringBuyMutation,
  useRevokeAccountAPIKeyMutation,
  useRevokePortfolioShareMutation,
  useRevokeWatchlistShareMutation,
  useSharePortfolioMutation,
  useShareWatchlistMutation,
  useStartExportMutation,
} from '@/libs/api';
import { setBrowserApiToken } from '@/libs/utils';
import { createdKeySessionToken, currentClientId, exportDownloadHref } from './SettingsPage.helpers';
import { FormRow, PageStack, Section } from './SettingsPage.styles';

export function SettingsPage() {
  const { t } = useTranslation(['settings', 'common']);
  const clientId = currentClientId();

  return (
    <PageStack>
      <PageHeader title={t('settings:title')} subtitle={t('settings:subtitle')} />
      <Text variant="caption" color="secondary">
        {t('settings:clientId')}: <Typography.Text copyable={{ text: clientId }}>{clientId}</Typography.Text>
      </Text>
      <Tabs
        items={[
          { key: 'keys', label: t('settings:tabs.keys'), children: <KeysPane /> },
          { key: 'export', label: t('settings:tabs.export'), children: <ExportPane /> },
          { key: 'sharing', label: t('settings:tabs.sharing'), children: <SharingPane /> },
          { key: 'recurring', label: t('settings:tabs.recurring'), children: <RecurringPane /> },
        ]}
      />
    </PageStack>
  );
}

function KeysPane() {
  const { t } = useTranslation(['settings', 'common']);
  const list = useListAccountAPIKeysQuery();
  const [create, createState] = useCreateAccountAPIKeyMutation();
  const [revoke] = useRevokeAccountAPIKeyMutation();
  const [secret, setSecret] = useState<string | null>(null);

  return (
    <Section>
      <Text variant="caption" color="secondary">
        {t('settings:keys.hint')}
      </Text>
      <Form
        layout="inline"
        onFinish={async (v: { name: string; permission: 'read' | 'trade' }) => {
          const got = await create(v).unwrap();
          setSecret(got.secret ?? null);
          const session = createdKeySessionToken(got.secret, got.permission ?? v.permission);
          if (session) setBrowserApiToken(session);
          void message.success(t('settings:keys.created'));
        }}
      >
        <Form.Item name="name" rules={[{ required: true }]} label={t('settings:keys.name')}>
          <Input maxLength={64} />
        </Form.Item>
        <Form.Item name="permission" initialValue="read" label={t('settings:keys.permission')}>
          <Select
            style={{ minWidth: 120 }}
            options={[
              { value: 'read', label: t('settings:keys.read') },
              { value: 'trade', label: t('settings:keys.trade') },
            ]}
          />
        </Form.Item>
        <Form.Item>
          <Button htmlType="submit" type="primary" loading={createState.isLoading}>
            {t('settings:keys.create')}
          </Button>
        </Form.Item>
      </Form>
      {secret ? (
        <Alert type="warning" showIcon message={t('settings:keys.created')} description={secret} />
      ) : null}
      {list.isError ? (
        <Alert type="error" showIcon message={t('settings:keys.loadFailed')} description={rtkErrorMessage(list.error)} />
      ) : null}
      <Table
        size="small"
        rowKey={(r) => r.id ?? r.prefix ?? ''}
        pagination={false}
        dataSource={list.data?.keys ?? []}
        locale={{ emptyText: t('settings:keys.empty') }}
        columns={[
          { title: t('settings:keys.name'), dataIndex: 'name' },
          { title: t('settings:keys.permission'), dataIndex: 'permission' },
          { title: t('settings:keys.prefix'), dataIndex: 'prefix' },
          {
            title: '',
            key: 'revoke',
            render: (_, row) =>
              row.id ? (
                <Button size="small" danger onClick={() => void revoke({ id: row.id as string })}>
                  {t('settings:keys.revoke')}
                </Button>
              ) : null,
          },
        ]}
      />
    </Section>
  );
}

function ExportPane() {
  const { t } = useTranslation(['settings', 'common']);
  const list = useListExportsQuery();
  const [start, startState] = useStartExportMutation();
  const [cancel] = useCancelExportMutation();

  return (
    <Section>
      <Text variant="caption" color="secondary">
        {t('settings:export.hint')}
      </Text>
      <Form
        layout="inline"
        onFinish={async (v: { format: 'json' | 'csv' }) => {
          await start({ format: v.format }).unwrap();
          void message.success(t('settings:export.started'));
        }}
      >
        <Form.Item name="format" initialValue="json" label={t('settings:export.format')}>
          <Select
            style={{ minWidth: 100 }}
            options={[
              { value: 'json', label: 'JSON' },
              { value: 'csv', label: 'CSV' },
            ]}
          />
        </Form.Item>
        <Form.Item>
          <Button htmlType="submit" type="primary" loading={startState.isLoading}>
            {t('settings:export.start')}
          </Button>
        </Form.Item>
      </Form>
      {list.isError ? (
        <Alert type="error" showIcon message={t('settings:export.loadFailed')} description={rtkErrorMessage(list.error)} />
      ) : null}
      <Table
        size="small"
        rowKey={(r) => r.id ?? r.createdAt ?? ''}
        pagination={false}
        dataSource={list.data?.exports ?? []}
        locale={{ emptyText: t('settings:export.empty') }}
        columns={[
          { title: 'id', dataIndex: 'id' },
          { title: t('settings:export.format'), dataIndex: 'format' },
          { title: 'status', dataIndex: 'status' },
          { title: '%', dataIndex: 'progressPct' },
          {
            title: '',
            key: 'act',
            render: (_, row) => {
              const href = exportDownloadHref(row.downloadUrl);
              return (
                <FormRow>
                  {href && row.status === 'completed' ? (
                    <Button size="small" href={href} target="_blank">
                      {t('settings:export.download')}
                    </Button>
                  ) : null}
                  {row.id && (row.status === 'pending' || row.status === 'running') ? (
                    <Button size="small" onClick={() => void cancel({ id: row.id as string })}>
                      {t('settings:export.cancel')}
                    </Button>
                  ) : null}
                </FormRow>
              );
            },
          },
        ]}
      />
    </Section>
  );
}

function SharingPane() {
  const { t } = useTranslation(['settings', 'common']);
  const wl = useListWatchlistSharesQuery();
  const pf = useListPortfolioSharesQuery();
  const [shareWl] = useShareWatchlistMutation();
  const [revWl] = useRevokeWatchlistShareMutation();
  const [sharePf] = useSharePortfolioMutation();
  const [revPf] = useRevokePortfolioShareMutation();

  return (
    <Section>
      <Text variant="label" color="primary">
        {t('settings:sharing.watchlist')}
      </Text>
      <Form
        layout="inline"
        onFinish={async (v: { granteeClientId: string; role: 'viewer' | 'editor' }) => {
          await shareWl(v).unwrap();
          void message.success(t('settings:sharing.share'));
        }}
      >
        <Form.Item name="granteeClientId" rules={[{ required: true }]} label={t('settings:sharing.grantee')}>
          <Input />
        </Form.Item>
        <Form.Item name="role" initialValue="viewer" label={t('settings:sharing.role')}>
          <Select
            style={{ minWidth: 120 }}
            options={[
              { value: 'viewer', label: t('settings:sharing.viewer') },
              { value: 'editor', label: t('settings:sharing.editor') },
            ]}
          />
        </Form.Item>
        <Form.Item>
          <Button htmlType="submit" type="primary">
            {t('settings:sharing.share')}
          </Button>
        </Form.Item>
      </Form>
      {wl.isError ? (
        <Alert type="error" showIcon message={t('settings:sharing.loadFailed')} description={rtkErrorMessage(wl.error)} />
      ) : null}
      <Table
        size="small"
        rowKey={(r) => r.granteeClientId ?? ''}
        pagination={false}
        dataSource={wl.data?.shares ?? []}
        locale={{ emptyText: t('settings:sharing.emptyWl') }}
        columns={[
          { title: t('settings:sharing.grantee'), dataIndex: 'granteeClientId' },
          { title: t('settings:sharing.role'), dataIndex: 'role' },
          {
            title: '',
            key: 'rev',
            render: (_, row) =>
              row.granteeClientId ? (
                <Button size="small" danger onClick={() => void revWl({ granteeClientId: row.granteeClientId as string })}>
                  {t('settings:sharing.revoke')}
                </Button>
              ) : null,
          },
        ]}
      />

      <Text variant="label" color="primary">
        {t('settings:sharing.portfolio')}
      </Text>
      <Form
        layout="inline"
        onFinish={async (v: { granteeClientId: string; role: 'viewer' | 'trader' }) => {
          await sharePf(v).unwrap();
          void message.success(t('settings:sharing.share'));
        }}
      >
        <Form.Item name="granteeClientId" rules={[{ required: true }]} label={t('settings:sharing.grantee')}>
          <Input />
        </Form.Item>
        <Form.Item name="role" initialValue="viewer" label={t('settings:sharing.role')}>
          <Select
            style={{ minWidth: 120 }}
            options={[
              { value: 'viewer', label: t('settings:sharing.viewer') },
              { value: 'trader', label: t('settings:sharing.trader') },
            ]}
          />
        </Form.Item>
        <Form.Item>
          <Button htmlType="submit" type="primary">
            {t('settings:sharing.share')}
          </Button>
        </Form.Item>
      </Form>
      {pf.isError ? (
        <Alert type="error" showIcon message={t('settings:sharing.loadFailed')} description={rtkErrorMessage(pf.error)} />
      ) : null}
      <Table
        size="small"
        rowKey={(r) => r.granteeClientId ?? ''}
        pagination={false}
        dataSource={pf.data?.shares ?? []}
        locale={{ emptyText: t('settings:sharing.emptyPf') }}
        columns={[
          { title: t('settings:sharing.grantee'), dataIndex: 'granteeClientId' },
          { title: t('settings:sharing.role'), dataIndex: 'role' },
          {
            title: '',
            key: 'rev',
            render: (_, row) =>
              row.granteeClientId ? (
                <Button size="small" danger onClick={() => void revPf({ granteeClientId: row.granteeClientId as string })}>
                  {t('settings:sharing.revoke')}
                </Button>
              ) : null,
          },
        ]}
      />
    </Section>
  );
}

function RecurringPane() {
  const { t } = useTranslation(['settings', 'common']);
  const list = useListRecurringBuysQuery();
  const [create, createState] = useCreateRecurringBuyMutation();
  const [pause] = usePauseRecurringBuyMutation();
  const [resume] = useResumeRecurringBuyMutation();
  const [del] = useDeleteRecurringBuyMutation();

  return (
    <Section>
      <Text variant="caption" color="secondary">
        {t('settings:recurring.hint')}
      </Text>
      <Form
        layout="inline"
        onFinish={async (v: { symbol: string; amount: number; frequency: 'daily' | 'weekly' | 'monthly' }) => {
          await create({
            symbol: v.symbol.trim().toUpperCase(),
            amount: Number(v.amount),
            frequency: v.frequency,
          }).unwrap();
          void message.success(t('settings:recurring.create'));
        }}
      >
        <Form.Item name="symbol" rules={[{ required: true }]} label={t('settings:recurring.symbol')}>
          <Input placeholder="BTCUSDT" />
        </Form.Item>
        <Form.Item name="amount" rules={[{ required: true }]} label={t('settings:recurring.amount')}>
          <InputNumber min={1} />
        </Form.Item>
        <Form.Item name="frequency" initialValue="daily" label={t('settings:recurring.frequency')}>
          <Select
            style={{ minWidth: 120 }}
            options={[
              { value: 'daily', label: t('settings:recurring.daily') },
              { value: 'weekly', label: t('settings:recurring.weekly') },
              { value: 'monthly', label: t('settings:recurring.monthly') },
            ]}
          />
        </Form.Item>
        <Form.Item>
          <Button htmlType="submit" type="primary" loading={createState.isLoading}>
            {t('settings:recurring.create')}
          </Button>
        </Form.Item>
      </Form>
      {list.isError ? (
        <Alert type="error" showIcon message={t('settings:recurring.loadFailed')} description={rtkErrorMessage(list.error)} />
      ) : null}
      <Table
        size="small"
        rowKey={(r) => r.id ?? ''}
        pagination={false}
        dataSource={list.data?.plans ?? []}
        locale={{ emptyText: t('settings:recurring.empty') }}
        columns={[
          { title: t('settings:recurring.symbol'), dataIndex: 'symbol' },
          { title: t('settings:recurring.amount'), dataIndex: 'amount' },
          { title: t('settings:recurring.frequency'), dataIndex: 'frequency' },
          { title: 'status', dataIndex: 'status' },
          { title: 'next', dataIndex: 'nextRunAt' },
          {
            title: '',
            key: 'act',
            render: (_, row) =>
              row.id ? (
                <FormRow>
                  {row.status === 'paused' ? (
                    <Button size="small" onClick={() => void resume({ id: row.id as string })}>
                      {t('settings:recurring.resume')}
                    </Button>
                  ) : (
                    <Button size="small" onClick={() => void pause({ id: row.id as string })}>
                      {t('settings:recurring.pause')}
                    </Button>
                  )}
                  <Button size="small" danger onClick={() => void del({ id: row.id as string })}>
                    {t('settings:recurring.delete')}
                  </Button>
                </FormRow>
              ) : null,
          },
        ]}
      />
    </Section>
  );
}
