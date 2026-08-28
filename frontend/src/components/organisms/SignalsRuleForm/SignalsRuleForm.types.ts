import type { CreateScannerRuleArg, ScannerRule } from '@/libs/api';

export type SignalsRuleFormProps = {
  intervals: string[];
  defaultInterval?: string;
  initialRule?: ScannerRule;
  isSubmitting?: boolean;
  submitError?: unknown;
  onSubmit: (values: CreateScannerRuleArg) => Promise<void>;
  onCancel?: () => void;
};
