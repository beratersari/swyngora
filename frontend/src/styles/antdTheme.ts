import type { ThemeConfig } from 'antd';
import { semanticColors, fontFamilies, radii } from '@/styles/tokens';

/**
 * Ant Design theme derived from Swyngora brand tokens.
 * Custom dark green palette — do not use default darkAlgorithm (washes out brand).
 */
export const antdTheme: ThemeConfig = {
  token: {
    colorPrimary: semanticColors.action.primary,
    colorInfo: semanticColors.accent.default,
    colorSuccess: semanticColors.status.success,
    colorWarning: semanticColors.status.warning,
    colorError: semanticColors.status.error,
    colorTextBase: semanticColors.text.primary,
    colorBgBase: semanticColors.bg.canvas,
    colorBgContainer: semanticColors.bg.muted,
    colorBgElevated: semanticColors.bg.elevated,
    colorBgLayout: semanticColors.bg.page,
    colorBorder: semanticColors.border.default,
    colorBorderSecondary: semanticColors.border.subtle,
    colorText: semanticColors.text.primary,
    colorTextSecondary: semanticColors.text.secondary,
    colorTextTertiary: semanticColors.text.tertiary,
    colorTextQuaternary: semanticColors.text.disabled,
    colorLink: semanticColors.text.link,
    colorLinkHover: semanticColors.text.linkHover,
    colorPrimaryHover: semanticColors.action.primaryHover,
    colorPrimaryActive: semanticColors.action.primaryActive,
    colorFillSecondary: semanticColors.bg.chrome,
    colorFillTertiary: semanticColors.bg.elevated,
    controlOutline: semanticColors.border.focus,
    controlItemBgHover: semanticColors.bg.hover,
    controlItemBgActive: semanticColors.bg.accentSoft,
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
      headerBg: semanticColors.bg.chrome,
      bodyBg: semanticColors.bg.page,
      footerBg: semanticColors.bg.canvas,
      siderBg: semanticColors.bg.muted,
      triggerBg: semanticColors.action.primary,
    },
    Card: {
      colorBgContainer: semanticColors.bg.muted,
      colorBorderSecondary: semanticColors.border.default,
    },
    Table: {
      headerBg: semanticColors.bg.tableHeader,
      rowHoverBg: semanticColors.bg.hover,
      borderColor: semanticColors.border.default,
      colorBgContainer: semanticColors.bg.muted,
      headerColor: semanticColors.text.primary,
      headerSplitColor: semanticColors.border.subtle,
      footerBg: semanticColors.bg.chrome,
    },
    Button: {
      primaryShadow: 'none',
      defaultBg: 'transparent',
      defaultBorderColor: semanticColors.action.secondaryBorder,
      defaultColor: semanticColors.text.primary,
      defaultHoverBorderColor: semanticColors.accent.default,
      defaultHoverColor: semanticColors.text.primary,
      defaultHoverBg: semanticColors.bg.hover,
      primaryColor: semanticColors.action.primaryText,
    },
    Input: {
      colorBgContainer: semanticColors.bg.chrome,
      colorBorder: semanticColors.border.default,
      activeBorderColor: semanticColors.border.focus,
      hoverBorderColor: semanticColors.accent.strong,
      colorText: semanticColors.text.primary,
      colorTextPlaceholder: semanticColors.text.tertiary,
    },
    Select: {
      colorBgContainer: semanticColors.bg.chrome,
      colorBgElevated: semanticColors.bg.elevated,
      optionSelectedBg: semanticColors.bg.accentSoft,
      optionActiveBg: semanticColors.bg.hover,
      colorBorder: semanticColors.border.default,
      colorText: semanticColors.text.primary,
      colorTextPlaceholder: semanticColors.text.tertiary,
    },
    Tabs: {
      // mountainMeadow — not caribbean neon for chrome
      inkBarColor: semanticColors.accent.default,
      itemSelectedColor: semanticColors.text.primary,
      itemColor: semanticColors.text.secondary,
      itemHoverColor: semanticColors.accent.default,
      itemActiveColor: semanticColors.text.primary,
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
    Tag: {
      // Never use Tag color="green"|"blue" presets on dark UI — they force light fills.
      defaultBg: semanticColors.bg.elevated,
      defaultColor: semanticColors.text.primary,
    },
    Alert: {
      // Fully dark-theme Alert — without color*Text Ant falls back to light surfaces
      // which yields white-on-white with colorTextBase = antiFlashWhite.
      colorInfoBg: semanticColors.bg.chrome,
      colorInfoBorder: semanticColors.border.accent,
      colorSuccessBg: semanticColors.bg.chrome,
      colorSuccessBorder: semanticColors.border.accent,
      colorWarningBg: withWarningBg(),
      colorWarningBorder: semanticColors.status.warning,
      colorErrorBg: semanticColors.bg.dangerSoft,
      colorErrorBorder: semanticColors.border.danger,
      colorIcon: semanticColors.accent.default,
      colorIconHover: semanticColors.accent.default,
      // Message body on all types
      colorText: semanticColors.text.primary,
      colorTextHeading: semanticColors.text.primary,
      colorTextDescription: semanticColors.text.secondary,
    },
    Pagination: {
      itemActiveBg: semanticColors.action.primary,
      colorPrimary: semanticColors.action.primary,
      colorPrimaryHover: semanticColors.action.primaryHover,
    },
    Empty: {
      colorText: semanticColors.text.secondary,
      colorTextDisabled: semanticColors.text.tertiary,
    },
    Message: {
      contentBg: semanticColors.bg.elevated,
    },
    Tooltip: {
      colorBgSpotlight: semanticColors.bg.elevated,
      colorTextLightSolid: semanticColors.text.primary,
    },
  },
};

function withWarningBg(): string {
  // soft warning wash from status.warning
  return 'rgba(224, 184, 106, 0.12)';
}
