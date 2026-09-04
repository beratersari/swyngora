import { Skeleton } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { Card, Section, Sections } from './ForensicsWhy.styles';
import type { ForensicsWhyProps } from './ForensicsWhy.types';

export function ForensicsWhy({ why, isLoading }: ForensicsWhyProps) {
  const { t } = useTranslation('forensics');
  if (isLoading) {
    return <Skeleton active paragraph={{ rows: 6 }} />;
  }
  if (!why?.headline && !why?.before && !why?.during) {
    return null;
  }
  const sections: { key: string; title: string; body?: string }[] = [
    { key: 'before', title: t('why.before'), body: why?.before },
    { key: 'during', title: t('why.during'), body: why?.during },
    { key: 'after', title: t('why.after'), body: why?.after },
    { key: 'similar', title: t('why.similar'), body: why?.similar },
  ];
  return (
    <Card aria-label={t('why.title')}>
      <Text variant="bodySm">{t('why.title')}</Text>
      {why?.headline ? <Text variant="body">{why.headline}</Text> : null}
      <Sections>
        {sections.map((sec) =>
          sec.body ? (
            <Section key={sec.key}>
              <Text variant="caption" color="secondary">
                {sec.title}
              </Text>
              <Text variant="bodySm">{sec.body}</Text>
            </Section>
          ) : null,
        )}
      </Sections>
      {why?.caution ? (
        <Text variant="caption" color="secondary">
          {why.caution}
        </Text>
      ) : null}
    </Card>
  );
}
