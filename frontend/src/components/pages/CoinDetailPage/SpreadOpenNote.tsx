import { Text } from '@/components/atoms/Text';
import { useGetWideSpreadQuery } from '@/libs/api';
import { rtkCurrent } from '@/libs/utils';

export function SpreadOpenNote({ exchange, symbol }: { exchange: string; symbol: string }) {
  const query = useGetWideSpreadQuery({ exchange, symbol }, { pollingInterval: 20_000 });
  const data = rtkCurrent(query);
  if (!data?.opened || !data.summary) return null;
  return (
    <Text variant="caption" color="secondary">
      {data.summary}
    </Text>
  );
}
