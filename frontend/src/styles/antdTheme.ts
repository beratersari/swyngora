import type { ThemeConfig } from 'antd';
import { colors, semanticColors, fontFamilies, radii } from '@/styles/tokens';

/** Ant Design theme derived from Swyngora design tokens */
export const antdTheme: ThemeConfig = {
  // Custom dark palette — do not use default darkAlgorithm (wrong blues)
  token: {
    colorPrimary: colors.indigo,
    colorInfo: colors.steel,
    colorSuccess: semanticColors.status.success,
    colorWarning: semanticColors.status.warning,
    colorError: semanticColors.status.error,
    colorTextBase: colors.cream,
    colorBgBase: colors.navy,
    colorBgContainer: semanticColors.bg.muted,
    colorBgElevated: colors.indigo,
    colorBgLayout: colors.navy,
    colorBorder: semanticColors.border.default,
    colorBorderSecondary: 'rgba(114, 136, 174, 0.2)',
    colorText: semanticColors.text.primary,
    colorTextSecondary: semanticColors.text.secondary,
    colorTextTertiary: semanticColors.text.disabled,
    colorTextQuaternary: semanticColors.text.disabled,
    colorLink: colors.cream,
    colorLinkHover: colors.steel,
    borderRadius: radii.md,
    borderRadiusLG: radii.lg,
    borderRadiusSM: radii.sm,
    fontFamily: fontFamilies.sans,
    fontFamilyCode: fontFamilies.mono,
    fontSize: 14,
    controlHeight: 36,
  },
  components: {
    Layout: {
      headerBg: colors.navy,
      bodyBg: colors.navy,
      footerBg: colors.navy,
      siderBg: semanticColors.bg.muted,
      triggerBg: colors.indigo,
    },
    Card: {
      colorBgContainer: semanticColors.bg.muted,
      colorBorderSecondary: semanticColors.border.default,
    },
    Table: {
      headerBg: 'rgba(75, 86, 148, 0.35)',
      rowHoverBg: 'rgba(75, 86, 148, 0.25)',
      borderColor: semanticColors.border.default,
    },
    Button: {
      primaryShadow: 'none',
      defaultBg: 'transparent',
      defaultBorderColor: colors.steel,
      defaultColor: colors.cream,
    },
    Skeleton: {
      gradientFromColor: semanticColors.skeleton.base,
      gradientToColor: semanticColors.skeleton.highlight,
    },
    Typography: {
      colorText: semanticColors.text.primary,
      colorTextDescription: semanticColors.text.secondary,
      colorTextHeading: semanticColors.text.primary,
    },
  },
};
