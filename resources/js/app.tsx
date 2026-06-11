import '../css/app.css';
import 'antd/dist/reset.css';

import {
  App as AntdApp,
  ConfigProvider,
} from 'antd';
import { resolvePageComponent } from 'laravel-vite-plugin/inertia-helpers';
import { createRoot } from 'react-dom/client';

import { createInertiaApp } from '@inertiajs/react';

createInertiaApp({
  resolve: (name) =>
    resolvePageComponent(`./Pages/${name}.tsx`, import.meta.glob('./Pages/**/*.tsx')),
  setup({ el, App, props }) {
    createRoot(el).render(
      <ConfigProvider
        theme={{
          token: { colorPrimary: '#1677ff', borderRadius: 6 },
        }}

      >
        <AntdApp>
          <App  {...props} />
        </AntdApp>
      </ConfigProvider>,
    );
  },
});
