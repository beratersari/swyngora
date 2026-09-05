import { Alert, Segmented } from 'antd';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { OptionsExpiryBars } from '@/components/organisms/OptionsExpiryBars';
import { OptionsStatCards } from '@/components/organisms/OptionsStatCards';
import { rtkErrorMessage, useGetMarketOptionsQuery } from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { rtkCurrent } from '@/libs/utils';
import { OPTIONS_ASSETS, OPTIONS_POLL_MS } from './constants';
import { parseOptionsAsset } from './helpers';
import { PageStack, Panel, Toolbar } from './OptionsPage.styles';

export function OptionsPage() {
  const { t } = useTranslation(['options', 'common']);
  const [searchParams, setSearchParams] = useSearchParams();
  const visible = useDocumentVisible();
  const asset = parseOptionsAsset(searchParams.get('asset'));

  const query = useGetMarketOptionsQuery(
    { asset },
    { pollingInterval: visible ? OPTIONS_POLL_MS : 0, refetchOnFocus: true },
  );
  const data = rtkCurrent(query);

  return (
    <PageStack>
      <PageHeader
        title={t('title')}
        subtitle={t('subtitle')}
        extra={
          <Segmented
            value={asset}
            options={OPTIONS_ASSETS.map((value) => ({ value, label: value }))}
            onChange={(next) => {
              setSearchParams(
                (prev) => {
                  const p = new URLSearchParams(prev);
                  p.set('asset', String(next));
                  return p;
                },
                { replace: true },
              );
            }}
            aria-label={t('assetAria')}
          />
        }
      />

      {query.isError ? (
        <Alert type="error" showIcon message={rtkErrorMessage(query.error)} />
      ) : null}

      <OptionsStatCards data={data} isLoading={query.isLoading} />

      <Panel>
        <Toolbar>
          <Text variant="bodySm" color="primary">
            {t('expiry.title')}
          </Text>
          <Text variant="caption" color="secondary">
            {t('expiry.legend')}
          </Text>
        </Toolbar>
        <OptionsExpiryBars expiries={data?.expiries} isLoading={query.isLoading} />
      </Panel>

      <Text variant="caption" color="secondary">
        {data?.note ?? t('disclaimer')}
      </Text>
    </PageStack>
  );
}
