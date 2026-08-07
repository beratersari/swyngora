import type { CreateScannerRuleArg, ScannerRuleType } from '@/libs/api';

export type SignalsRuleFormProps = {
  intervals: string[];
  defaultInterval?: string;
  isSubmitting?: boolean;
  submitError?: unknown;
  onSubmit: (values: CreateScannerRuleArg) => Promise<void>;
};

export type SignalsRuleFormValues = CreateScannerRuleArg & {
  type: ScannerRuleType;
};
