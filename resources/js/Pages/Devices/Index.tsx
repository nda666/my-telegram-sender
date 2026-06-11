import {
  useEffect,
  useState,
} from 'react';

import {
  Button,
  Dropdown,
  MenuProps,
  Popconfirm,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import { ColumnsType } from 'antd/es/table';

import {
  DeleteOutlined,
  EditOutlined,
  KeyOutlined,
  MessageOutlined,
  MoreOutlined,
  PlusOutlined,
  SyncOutlined,
  UserOutlined,
} from '@ant-design/icons';
import {
  Link,
  router,
  usePage,
} from '@inertiajs/react';

import AppLayout from '../../Layouts/AppLayout';

type Device = {
  id: number;
  name: string;
  phone: string;
  telegramFirstName: string;
  telegramLastName: string;
  telegramPhone: string;
  status: 'no_session' | 'offline' | 'online';
  hasSession: boolean;
  createdAt: string;
};

type PageProps = {
  devices: Device[];
};

const statusConfig = {
  no_session: { color: 'default', label: 'Belum ada session' },
  offline: { color: 'orange', label: 'Offline' },
  online: { color: 'green', label: 'Online' },
} as const;

// Satu SSE connection per device yang punya session
function useDevicesStatus(devices: Device[]) {
  const [statusMap, setStatusMap] = useState<Record<number, Device['status']>>(() =>
    Object.fromEntries(devices.map((d) => [d.id, d.status]))
  );

  useEffect(() => {
    const sources: EventSource[] = [];

    devices
      .filter((d) => d.hasSession)
      .forEach((d) => {
        const es = new EventSource(`/devices/${d.id}/status/stream`);
        es.onmessage = (e) => {
          const s = e.data.trim() as Device['status'];
          setStatusMap((prev) => ({ ...prev, [d.id]: s }));
        };
        sources.push(es);
      });

    return () => sources.forEach((es) => es.close());
  }, []); // hanya mount sekali

  return statusMap;
}

export default function DevicesIndex() {
  const { devices } = usePage<PageProps>().props;
  const [refreshingProfile, setRefreshingProfile] = useState(false);
  const statusMap = useDevicesStatus(devices);

  const refreshProfile = async (deviceId: number) => {
    setRefreshingProfile(true);
    const res = await fetch(`/devices/${deviceId}/profile`);
    const data = await res.json();
    if (!data.error) {
      router.reload({ only: ['devices'] }); // Inertia partial reload

    }
    setRefreshingProfile(false)
  };

  const columns: ColumnsType<Device> = [
    { title: 'Nama', dataIndex: 'name', key: 'name' },
    { title: 'Phone', dataIndex: 'phone', key: 'phone' },
    {
      title: 'Telegram',
      key: 'telegram',
      render: (_: unknown, row: Device) =>
        row.telegramFirstName ? `${row.telegramFirstName} ${row.telegramLastName} (${row.telegramPhone})` : '-',
    },
    { title: 'Api Key', dataIndex: 'apiKey', key: 'apiKey' },
    {
      title: 'Status',
      key: 'status',
      render: (_: unknown, row: Device) => {
        const status = statusMap[row.id] ?? row.status;
        const cfg = statusConfig[status] ?? statusConfig.no_session;
        const isOnline = status === 'online';
        return (
          <Tag
            color={cfg.color}
          // icon={isOnline ? <SyncOutlined spin /> : undefined}
          >
            {cfg.label}
          </Tag>
        );
      },
    },
    { title: 'Dibuat', dataIndex: 'createdAt', key: 'createdAt' },
    {
      title: 'Aksi',
      key: 'actions',
      render: (_, row) => {
        const items: MenuProps['items'] = [
          {
            key: 'edit',
            icon: <EditOutlined />,
            label: (
              <Link href={`/devices/${row.id}/edit`}>
                Edit
              </Link>
            ),
          },
          { type: 'divider' },
          {
            key: 'session',
            icon: <KeyOutlined />,
            label: (
              <Link href={`/devices/${row.id}/session`}>
                Session
              </Link>
            ),
          },
        ];

        if (row.hasSession) {
          items.push(
            {
              key: 'inbox',
              icon: <MessageOutlined />,
              label: (
                <Link href={`/devices/${row.id}/inbox`}>
                  Inbox
                </Link>
              ),
            },
            {
              key: 'contacts',
              icon: <UserOutlined />,
              label: (
                <Link href={`/devices/${row.id}/contacts`}>
                  Contacts
                </Link>
              ),
            },
            {
              key: 'refresh',
              icon: <SyncOutlined />,
              label: <>'Refresh Profile'</>,
              onClick: () => refreshProfile(row.id),
            },
          );
        }

        items.push({ type: 'divider' });

        items.push({
          key: 'delete',
          icon: <DeleteOutlined />,
          danger: true,
          label: (
            <Popconfirm
              title="Hapus device ini?"
              onConfirm={() => router.delete(`/devices/${row.id}`)}
            >
              Hapus
            </Popconfirm>
          ),
        });

        return (
          <Space>
            {row.hasSession ? (
              <Link href={`/devices/${row.id}/inbox`}>
                <Button
                  size="small"
                  type="primary"
                  icon={<MessageOutlined />}
                >
                  Inbox
                </Button>
              </Link>
            ) : (
              <Link href={`/devices/${row.id}/session`}>
                <Button
                  size="small"
                  type="primary"
                  icon={<KeyOutlined />}
                >
                  Session
                </Button>
              </Link>
            )}
            <Dropdown
              menu={{ items }}
              trigger={['click']}
            >
              <Button
                size="small"
                icon={<MoreOutlined />}
              >
                Lainnya
              </Button>
            </Dropdown>
          </Space>
        );
      },
    },
  ];

  return (
    <AppLayout>
      <Space style={{ marginBottom: 16, width: '100%', justifyContent: 'space-between' }}>
        <Typography.Title level={4} style={{ margin: 0 }}>
          Devices
        </Typography.Title>
        <Link href="/devices/create">
          <Button type="primary" icon={<PlusOutlined />}>Tambah Device</Button>
        </Link>
      </Space>
      <Table rowKey="id" dataSource={devices} columns={columns} pagination={false} />
    </AppLayout>
  );
}