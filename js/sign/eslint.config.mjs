// @ts-check

import eslint from '@eslint/js';
import { defineConfig } from 'eslint/config';
import globals from 'globals';
import tseslint from 'typescript-eslint';

export default defineConfig(
  {
    ignores: ['lib/**', 'node_modules/**'],
  },
  eslint.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ['**/*.ts', '**/*.js'],
    languageOptions: {
      globals: {
        ...globals.node,
      },
    },
  },
  // Test specific rules/globals
  {
    files: ['**/tests/**/*.js', '**/*_test.js'],
    languageOptions: {
      globals: {
        ...globals.jasmine,
      },
    },
  }
);
