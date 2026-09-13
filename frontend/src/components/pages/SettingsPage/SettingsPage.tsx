import { useState } from 'react';
import { Alert, Button, Form, Input, InputNumber, Radio, Select, Table, Tabs, Typography, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import {
  getRtkErrorCode,
  rtkErrorMessage,
  useCancelExportMutation,
  useCancelImportMutation,
  useConfirmImportMutation,
  useCreateAccountAPIKeyMutation,
  useCompleteRecurringBuyRunMutation,
  useCreateRecurringBuyMutation,
  useDeleteRecurringBuyMutation,
  useListAccountAPIKeysQuery,
  useListExportsQuery,
  useListImportsQuery,
  useListPortfolioSharesQuery,
  useListRecurringBuyRunsQuery,
  useListRecurringBuysQuery,
  useListWatchlistSharesQuery,
  usePauseRecurringBuyMutation,
  usePreviewImportMutation,
  useResumeRecurringBuyMutation,
  useRevokeAccountAPIKeyMutation,
  useRevokePortfolioShareMutation,
  useRevokeWatchlistShareMutation,
  useSharePortfolioMutation,
  useShareWatchlistMutation,
  useStartExportMutation,
  type ImportJob,
} from '@/libs/api';
import { setBrowserApiToken } from '@/libs/utils';
import {
  createdKeySessionToken,
  currentClientId,
  customSignalImportPreview,
  customSignalResolutionsFromPicks,
  exportDownloadHref,
  type CustomSignalConflictPick,
} from './SettingsPage.helpers';
import { ConflictCard, FormRow, NameList, PageStack, Section } from './SettingsPage.styles';

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
          { key: 'import', label: t('settings:tabs.import'), children: <ImportPane /> },
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

function ImportPane() {
  const { t } = useTranslation(['settings', 'common']);
  const list = useListImportsQuery(undefined, { pollingInterval: 4000 });
  const [preview, previewState] = usePreviewImportMutation();
  const [confirm, confirmState] = useConfirmImportMutation();
  const [cancel] = useCancelImportMutation();
  const [file, setFile] = useState<File | null>(null);
  const [job, setJob] = useState<ImportJob | null>(null);
  const [mode, setMode] = useState<'merge' | 'replace'>('merge');
  const [picks, setPicks] = useState<Record<string, CustomSignalConflictPick>>({});
  const [stale, setStale] = useState(false);
  const profiles = customSignalImportPreview(job);

  return (
    <Section>
      <Text variant="caption" color="secondary">
        {t('settings:import.hint')}
      </Text>
      <FormRow>
        <input
          type="file"
          accept=".json,.csv,application/json,text/csv"
          data-testid="import-file"
          onChange={(e) => {
            setFile(e.target.files?.[0] ?? null);
            setJob(null);
            setPicks({});
            setStale(false);
          }}
        />
        <Button
          type="primary"
          disabled={!file}
          loading={previewState.isLoading}
          onClick={async () => {
            if (!file) return;
            const got = await preview({ file }).unwrap();
            setJob(got);
            setPicks({});
            setStale(false);
            void message.success(t('settings:import.previewed'));
          }}
        >
          {t('settings:import.preview')}
        </Button>
      </FormRow>
      {stale ? (
        <Alert type="warning" showIcon message={t('settings:import.stalePreview')} data-testid="import-stale" />
      ) : null}
      {previewState.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('settings:import.previewFailed')}
          description={rtkErrorMessage(previewState.error)}
        />
      ) : null}
      {job ? (
        <>
          <Text variant="bodySm" data-testid="import-preview-counts">
            {t('settings:import.counts', {
              valid: job.totals?.valid ?? 0,
              willAdd: job.totals?.willAdd ?? 0,
              willUpdate: job.totals?.willUpdate ?? 0,
              willConflict: job.totals?.willConflict ?? 0,
            })}
          </Text>
          <Text variant="label" color="primary">
            {t('settings:import.profilesAdd')}
          </Text>
          <NameList data-testid="import-preview-add">
            {profiles.addNames.length === 0 ? (
              <li>{t('settings:import.none')}</li>
            ) : (
              profiles.addNames.map((name) => <li key={name}>{name}</li>)
            )}
          </NameList>
          <Text variant="label" color="primary">
            {t('settings:import.profilesUpdate')}
          </Text>
          <NameList data-testid="import-preview-update">
            {profiles.updateNames.length === 0 ? (
              <li>{t('settings:import.none')}</li>
            ) : (
              profiles.updateNames.map((name) => <li key={name}>{name}</li>)
            )}
          </NameList>
          <Text variant="label" color="primary">
            {t('settings:import.profilesConflict')}
          </Text>
          {profiles.conflicts.length === 0 ? (
            <NameList data-testid="import-preview-conflicts">
              <li>{t('settings:import.none')}</li>
            </NameList>
          ) : (
            <div data-testid="import-preview-conflicts">
              {profiles.conflicts.map((conflict) => {
                const name = conflict.name ?? '';
                const pick = picks[name] ?? {};
                const needIncoming = (conflict.incoming?.length ?? 0) > 1;
                const needTarget = (conflict.candidates?.length ?? 0) > 1;
                return (
                  <ConflictCard key={name} data-testid={`import-conflict-${name}`}>
                    <Text variant="label" color="primary">
                      {name}
                    </Text>
                    <Text variant="caption" color="secondary">
                      {t('settings:import.conflictHint')}
                    </Text>
                    {needIncoming ? (
                      <Radio.Group
                        value={pick.incomingIndex}
                        onChange={(e) =>
                          setPicks((cur) => ({ ...cur, [name]: { ...cur[name], incomingIndex: e.target.value as number } }))
                        }
                        options={(conflict.incoming ?? []).map((row) => ({
                          value: row.index ?? 0,
                          label: t('settings:import.incomingOption', {
                            index: (row.index ?? 0) + 1,
                            default: row.isDefault ? t('settings:import.defaultBadge') : '',
                          }),
                        }))}
                      />
                    ) : null}
                    {needTarget ? (
                      <Radio.Group
                        value={pick.profileId ?? ''}
                        onChange={(e) =>
                          setPicks((cur) => ({
                            ...cur,
                            [name]: { ...cur[name], profileId: String(e.target.value || '') || undefined },
                          }))
                        }
                        options={[
                          { value: '', label: t('settings:import.skip') },
                          ...(conflict.candidates ?? []).map((cand) => ({
                            value: cand.id ?? '',
                            label: t('settings:import.candidateOption', {
                              name: cand.name ?? cand.id,
                              id: (cand.id ?? '').slice(0, 8),
                              default: cand.isDefault ? t('settings:import.defaultBadge') : '',
                            }),
                          })),
                        ]}
                      />
                    ) : null}
                  </ConflictCard>
                );
              })}
            </div>
          )}
          <Form
            layout="inline"
            onFinish={async () => {
              if (!job.id) return;
              try {
                await confirm({
                  id: job.id,
                  mode,
                  customSignalResolutions: customSignalResolutionsFromPicks(profiles.conflicts, picks),
                }).unwrap();
                setStale(false);
                void message.success(t('settings:import.started'));
              } catch (err) {
                if (getRtkErrorCode(err) === 'import_preview_stale') {
                  setStale(true);
                  setJob(null);
                  setPicks({});
                  return;
                }
                void message.error(rtkErrorMessage(err));
              }
            }}
          >
            <Form.Item label={t('settings:import.mode')}>
              <Select
                style={{ minWidth: 140 }}
                value={mode}
                onChange={setMode}
                options={[
                  { value: 'merge', label: t('settings:import.merge') },
                  { value: 'replace', label: t('settings:import.replace') },
                ]}
              />
            </Form.Item>
            <Form.Item>
              <Button htmlType="submit" type="primary" loading={confirmState.isLoading}>
                {t('settings:import.confirm')}
              </Button>
            </Form.Item>
          </Form>
        </>
      ) : null}
      {list.isError ? (
        <Alert type="error" showIcon message={t('settings:import.loadFailed')} description={rtkErrorMessage(list.error)} />
      ) : null}
      <Table
        size="small"
        rowKey={(r) => r.id ?? r.createdAt ?? ''}
        pagination={false}
        dataSource={list.data?.imports ?? []}
        locale={{ emptyText: t('settings:import.empty') }}
        columns={[
          { title: 'id', dataIndex: 'id' },
          { title: t('settings:import.mode'), dataIndex: 'mode' },
          { title: 'status', dataIndex: 'status' },
          { title: '%', dataIndex: 'progressPct' },
          {
            title: '',
            key: 'act',
            render: (_, row) =>
              row.id && (row.status === 'previewed' || row.status === 'pending' || row.status === 'running') ? (
                <Button size="small" onClick={() => void cancel({ id: row.id as string })}>
                  {t('settings:import.cancel')}
                </Button>
              ) : null,
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
        expandable={{
          expandedRowRender: (row) => (row.id ? <RecurringRuns planId={row.id} /> : null),
        }}
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

function RecurringRuns({ planId }: { planId: string }) {
  const { t } = useTranslation(['settings', 'common']);
  const list = useListRecurringBuyRunsQuery({ id: planId });
  const [complete, completeState] = useCompleteRecurringBuyRunMutation();
  return (
    <Table
      size="small"
      rowKey={(r) => r.id ?? ''}
      pagination={false}
      loading={list.isLoading}
      dataSource={list.data?.runs ?? []}
      locale={{ emptyText: t('settings:recurring.noRuns') }}
      columns={[
        { title: t('settings:recurring.scheduled'), dataIndex: 'scheduledFor' },
        { title: t('settings:recurring.status'), dataIndex: 'status' },
        {
          title: t('settings:recurring.timing'),
          dataIndex: 'timing',
          render: (v: string | undefined) => {
            if (v === 'on_time') return t('settings:recurring.onTime');
            if (v === 'late') return t('settings:recurring.late');
            return '—';
          },
        },
        { title: t('settings:recurring.amount'), dataIndex: 'amount' },
        {
          title: '',
          key: 'act',
          render: (_, row) =>
            row.canComplete && row.id ? (
              <Button
                size="small"
                loading={completeState.isLoading}
                onClick={() => {
                  void complete({ planId, runId: row.id as string })
                    .unwrap()
                    .then(() => {
                      void message.success(t('settings:recurring.completed'));
                    })
                    .catch((err: unknown) => {
                      void message.error(rtkErrorMessage(err));
                    });
                }}
              >
                {t('settings:recurring.complete')}
              </Button>
            ) : null,
        },
      ]}
    />
  );
}
