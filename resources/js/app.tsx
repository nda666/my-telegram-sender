import '../css/app.css';
import 'antd/dist/reset.css';
import { createRoot } from 'react-dom/client';
import { createInertiaApp } from '@inertiajs/react';
import { resolvePageComponent } from 'laravel-vite-plugin/inertia-helpers';
import { ConfigProvider } from 'antd';

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
        <App {...props} />
      </ConfigProvider>,
    );
  },
});
