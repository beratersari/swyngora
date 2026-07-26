import type { ThemeConfig } from 'antd';
import { palette, semanticColors, fontFamilies, radii } from '@/styles/tokens';

/** Ant Design theme derived from Swyngora green design tokens */
export const antdTheme: ThemeConfig = {
  // Custom dark green palette — do not use default darkAlgorithm
  token: {
    colorPrimary: palette.bangladeshGreen,
    colorInfo: palette.frog,
    colorSuccess: semanticColors.status.success,
    colorWarning: semanticColors.status.warning,
    colorError: semanticColors.status.error,
    colorTextBase: palette.antiFlashWhite,
    colorBgBase: palette.richBlack,
    colorBgContainer: semanticColors.bg.muted,
    colorBgElevated: palette.basil,
    colorBgLayout: palette.richBlack,
    colorBorder: semanticColors.border.default,
    colorBorderSecondary: 'rgba(170, 203, 196, 0.18)',
    colorText: semanticColors.text.primary,
    colorTextSecondary: semanticColors.text.secondary,
    colorTextTertiary: semanticColors.text.disabled,
    colorTextQuaternary: semanticColors.text.disabled,
    colorLink: palette.mountainMeadow,
    colorLinkHover: palette.mint,
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
      headerBg: palette.richBlack,
      bodyBg: palette.richBlack,
      footerBg: palette.richBlack,
      siderBg: semanticColors.bg.muted,
      triggerBg: palette.bangladeshGreen,
    },
    Card: {
      colorBgContainer: semanticColors.bg.muted,
      colorBorderSecondary: semanticColors.border.default,
    },
    Table: {
      headerBg: 'rgba(3, 98, 76, 0.35)',
      rowHoverBg: 'rgba(23, 135, 109, 0.22)',
      borderColor: semanticColors.border.default,
    },
    Button: {
      primaryShadow: 'none',
      defaultBg: 'transparent',
      defaultBorderColor: palette.stone,
      defaultColor: palette.antiFlashWhite,
    },
    Tabs: {
      inkBarColor: palette.caribbeanGreen,
      itemSelectedColor: palette.antiFlashWhite,
      itemColor: palette.stone,
      itemHoverColor: palette.mint,
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
      defaultBg: palette.pine,
      defaultColor: palette.pistachio,
    },
  },
};
