/** The three ways a crypto market order can work one size. */
export const EXECUTION_STYLES = ['market', 'best_cost', 'spread'] as const;

export type ExecutionStyleId = (typeof EXECUTION_STYLES)[number];
