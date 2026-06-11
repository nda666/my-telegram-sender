import type { ReactNode } from 'react';

import {
  Button,
  Layout,
  Menu,
  Typography,
} from 'antd';

import {
  FileTextOutlined,
  KeyOutlined,
  LogoutOutlined,
  MessageOutlined,
  MobileOutlined,
  SettingOutlined,
  UserOutlined,
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

  // 1. Bersihkan url dari query string
  const pathname = url.split('?')[0];

  // 2. Pecah path menjadi array segment
  const segments = pathname.split('/').filter(Boolean);

  // 3. Cek apakah kita sedang berada di sub-halaman device (misal: /devices/1/session)
  // Kita cari tahu apakah segment pertama adalah 'devices' dan segment kedua adalah sebuah ID (Angka)
  const isDeviceSubPage = segments[0] === 'devices' && segments[1] && !isNaN(Number(segments[1]));
  const deviceId = isDeviceSubPage ? segments[1] : null;

  // 4. Bersihkan segmen dari ID Angka untuk pencocokan key Antd Menu
  const cleanSegments = segments.filter(seg => isNaN(Number(seg)));

  // 5. Generate keys potensial (Menghasilkan: ['devices-session', 'devices'])
  const potentialKeys = cleanSegments.reduce<string[]>((acc, _, index) => {
    const key = cleanSegments.slice(0, index + 1).join('-');
    acc.unshift(key);
    return acc;
  }, []);

  // 6. Susun struktur menu utama
  const menuItems: any[] = [
    {
      key: 'devices',
      icon: <MobileOutlined />,
      label: <Link href="/devices">Devices</Link>,
    },
  ];

  // LOGIKA DINAMIS: Jika sedang membuka device tertentu, suntikkan child menu di bawah Devices
  if (isDeviceSubPage && deviceId) {
    menuItems[0].children = [
      {
        key: 'devices', // Key ini agar saat klik nama induk/kembali ke list utama tetap aman
        label: <Link href="/devices">← Back to List</Link>,
      },
      {
        key: 'devices-session', // Gabungan clean segments: 'devices' + '-' + 'session'
        icon: <KeyOutlined />,
        label: <Link href={`/devices/${deviceId}/session`}>Session</Link>,
      },
      {
        key: 'devices-inbox',
        icon: <MessageOutlined />,
        label: <Link href={`/devices/${deviceId}/inbox`}>Inbox</Link>,
      },
      {
        key: 'devices-contacts',
        icon: <UserOutlined />,
        label: <Link href={`/devices/${deviceId}/contacts`}>Contacts</Link>,
      },
    ];
  }

  // Tambahkan menu lainnya setelah menu devices selesai diproses
  menuItems.push(
    {
      key: 'logs',
      icon: <FileTextOutlined />,
      label: <Link href="/logs">Logs</Link>,
    },
    {
      key: 'settings',
      icon: <SettingOutlined />,
      label: 'Settings',
      children: [
        {
          icon: <KeyOutlined />,
          key: 'settings-password',
          label: <Link href="/settings/password">Password</Link>,
        },
      ],
    },
  );

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
          selectedKeys={potentialKeys}
          defaultOpenKeys={['devices', 'settings']}
          items={menuItems}
        />
      </Sider>
      <Layout style={{ display: 'flex', flexDirection: 'column' }}>
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
        <Content
          style={{
            flex: 1,
            padding: 24,
            display: 'flex',
            flexDirection: 'column',
            overflow: 'auto',
          }}
        >
          {children}
        </Content>
      </Layout>
    </Layout>
  );
}