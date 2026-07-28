import js from '@eslint/js';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import tseslint from 'typescript-eslint';
import prettier from 'eslint-config-prettier';

export default tseslint.config(
  { ignores: ['dist', 'node_modules', 'src/libs/api/generated/**'] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended, prettier],
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2022,
      globals: globals.browser,
    },
    plugins: {
      'react-hooks': reactHooks,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
    },
  },
  // Boundary: components must not import modules or app
  {
    files: ['src/components/**/*.{ts,tsx}'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['@/modules', '@/modules/*', '@/app', '@/app/*'],
              message: 'Atomic components must not import modules or app (pages live in modules).',
            },
          ],
        },
      ],
    },
  },
  // Boundary: libs must not import modules, app, or components
  {
    files: ['src/libs/**/*.{ts,tsx}'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: [
                '@/modules',
                '@/modules/*',
                '@/app',
                '@/app/*',
                '@/components',
                '@/components/*',
              ],
              message: 'libs must not import modules, app, or UI components.',
            },
          ],
        },
      ],
    },
  },
);
