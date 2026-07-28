export const AiScreens = {
  Chat: 'AiChat',
} as const;

export type AiStackParamList = {
  AiChat:
    | {
        draft?: string;
        exchange?: string;
        symbol?: string;
        interval?: string;
      }
    | undefined;
};
