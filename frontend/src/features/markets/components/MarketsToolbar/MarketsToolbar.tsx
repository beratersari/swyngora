import { Input } from 'antd';
import { SearchOutlined } from '@ant-design/icons';
import { QUOTE_OPTIONS, SEARCH_PLACEHOLDER } from './MarketsToolbar.constants';
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
}: MarketsToolbarProps) {
  const tagOptions = [
    { value: '', label: 'All tags' },
    ...tags.map((t) => ({ value: t, label: t })),
  ];

  return (
    <ToolbarRow>
      <SearchWrap>
        <Input
          allowClear
          prefix={<SearchOutlined />}
          placeholder={SEARCH_PLACEHOLDER}
          value={q}
          onChange={(e) => onQChange(e.target.value)}
          aria-label="Search markets"
        />
      </SearchWrap>
      <FieldWrap>
        <QuoteSelect
          value={quote}
          options={[...QUOTE_OPTIONS]}
          onChange={(v) => onQuoteChange(String(v))}
          aria-label="Quote asset"
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
          aria-label="Product tag"
          placeholder="Tag"
        />
      </FieldWrap>
    </ToolbarRow>
  );
}
