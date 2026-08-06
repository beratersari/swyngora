import { Input } from 'antd';
import { SearchOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { QUOTE_OPTIONS } from './MarketsToolbar.constants';
import { FieldWrap, QuoteSelect, SearchWrap, TagSelect, ToolbarRow } from './MarketsToolbar.styles';
import type { MarketsToolbarProps } from './MarketsToolbar.types';

export function MarketsToolbar({
  q,
  quote,
  tag,
  tags,
  tagsLoading,
  onQChange,
  onQuoteChange,
  onTagChange,
  trailing,
}: MarketsToolbarProps) {
  const { t } = useTranslation('markets');

  // Put "Delist" first so it is obvious in the tag filter (not buried alphabetically).
  const sortedTags = [...tags].sort((a, b) => {
    const aD = a.toLowerCase() === 'delist' ? 0 : 1;
    const bD = b.toLowerCase() === 'delist' ? 0 : 1;
    if (aD !== bD) return aD - bD;
    return a.localeCompare(b);
  });
  const tagOptions = [
    { value: '', label: t('filters.allTags') },
    ...sortedTags.map((item) => ({
      value: item,
      label: item.toLowerCase() === 'delist' ? t('filters.delistTag') : item,
    })),
  ];

  return (
    <ToolbarRow>
      <SearchWrap>
        <Input
          allowClear
          prefix={<SearchOutlined />}
          placeholder={t('search.placeholder')}
          value={q}
          onChange={(e) => onQChange(e.target.value)}
          aria-label={t('search.ariaLabel')}
        />
      </SearchWrap>
      <FieldWrap>
        <QuoteSelect
          value={quote}
          options={[...QUOTE_OPTIONS]}
          onChange={(v) => onQuoteChange(String(v))}
          aria-label={t('filters.quoteAria')}
        />
      </FieldWrap>
      <FieldWrap>
        <TagSelect
          loading={tagsLoading}
          value={tag || ''}
          options={tagOptions}
          onChange={(v) => onTagChange(v != null ? String(v) : '')}
          showSearch
          optionFilterProp="label"
          aria-label={t('filters.tagAria')}
          placeholder={t('filters.tagPlaceholder')}
        />
      </FieldWrap>
      {trailing ? <FieldWrap>{trailing}</FieldWrap> : null}
    </ToolbarRow>
  );
}
