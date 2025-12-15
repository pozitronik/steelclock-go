export default [
  {
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'script',
      globals: {
        // Browser globals
        window: 'readonly',
        document: 'readonly',
        console: 'readonly',
        localStorage: 'readonly',
        fetch: 'readonly',
        WebSocket: 'readonly',
        navigator: 'readonly',
        performance: 'readonly',
        setTimeout: 'readonly',
        setInterval: 'readonly',
        clearInterval: 'readonly',
        clearTimeout: 'readonly',
        atob: 'readonly',
        confirm: 'readonly',
        prompt: 'readonly',
        Event: 'readonly',
        // App globals (classes defined in other files)
        API: 'writable',
        FormBuilder: 'writable',
        SchemaProcessor: 'writable',
        WidgetRegistry: 'writable',
        WidgetEditor: 'writable',
        previewPanel: 'writable',
        PreviewPanel: 'writable'
      }
    },
    rules: {
      'no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^(API|FormBuilder|SchemaProcessor|WidgetRegistry|WidgetEditor|PreviewPanel)$', caughtErrorsIgnorePattern: '^_' }],
      'semi': ['error', 'always'],
      'no-undef': 'error'
    }
  }
];
