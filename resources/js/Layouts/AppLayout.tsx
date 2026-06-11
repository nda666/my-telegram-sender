import type { ReactNode } from 'react';

import {
  Button,
  Layout,
  Menu,
  Typography,
} from 'antd';

import {
  FileTextOutlined,
  LogoutOutlined,
  MobileOutlined,
} from '@ant-design/icons';
import {
  Link,
  router,
  usePage,
} from '@inertiajs/react';

const { Header, Sider, Content } = Layout;

type AuthUser = {
  id: number;
  username: string;
  name: string;
};

type PageProps = {
  auth?: { user: AuthUser };
};

export default function AppLayout({ children }: { children: ReactNode }) {
  const { url, props } = usePage<PageProps>();
  const user = props.auth?.user;

  const selected = url.startsWith('/logs')
    ? 'logs'
    : url.startsWith('/devices')
      ? 'devices'
      : '';

  return (
    <Layout style={{ minHeight: '100vh', height: '100vh', overflow: 'hidden' }}>
      <Sider breakpoint="lg" collapsedWidth={0}>
        <div style={{ padding: '16px', textAlign: 'center' }}>
          <Typography.Title level={5} style={{ color: '#fff', margin: 0 }}>
            TG Sender
          </Typography.Title>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selected]}
          items={[
            {
              key: 'devices',
              icon: <MobileOutlined />,
              label: <Link href="/devices">Devices</Link>,
            },
            {
              key: 'logs',
              icon: <FileTextOutlined />,
              label: <Link href="/logs">Logs</Link>,
            },
          ]}
        />
      </Sider>
      <Layout style={{ display: 'flex', flexDirection: 'column', }}>
        <Header
          style={{
            background: '#fff',
            padding: '0 24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <Typography.Text type="secondary">
            {user ? `Halo, ${user.name || user.username}` : ''}
          </Typography.Text>
          <Button
            type="text"
            icon={<LogoutOutlined />}
            onClick={() => router.post('/logout')}
          >
            Logout
          </Button>
        </Header>
        <Content style={{
          flex: 1,
          padding: 24,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'auto'
        }}>{children}</Content>
      </Layout>
    </Layout>
  );
}
