import { useState } from 'react';

import {
  Button,
  Dropdown,
  Flex,
  MenuProps,
  Popconfirm,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import { ColumnsType } from 'antd/es/table';

import {
  ApiOutlined,
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
  hasSession: boolean;
  createdAt: string;
};

type PageProps = {
  devices: Device[];
};

const statusConfig = {
  no_session: { color: 'default', label: 'Belum ada session' },
  has_session: { color: 'green', label: 'Session OK' },
} as const;


export default function DevicesIndex() {
  const [deleteTarget, setDeleteTarget] = useState<Device | null>(null);
  const { devices } = usePage<PageProps>().props;
  const [refreshingProfile, setRefreshingProfile] = useState<Record<number, boolean>>({});
  const refreshProfile = async (deviceId: number) => {
    setRefreshingProfile(prev => ({
      ...prev,
      [deviceId]: true,
    }));
    const res = await fetch(`/devices/${deviceId}/profile`);
    const data = await res.json();
    if (!data.error) {
      router.reload({ only: ['devices'] }); // Inertia partial reload

    }
    setRefreshingProfile(prev => ({
      ...prev,
      [deviceId]: false,
    }));
  };

  const columns: ColumnsType<Device> = [
    { title: 'Nama', dataIndex: 'name', key: 'name' },
    { title: 'Phone', dataIndex: 'phone', key: 'phone' },
    {
      title: 'Telegram',
      key: 'telegram',
      render: (_: unknown, row: Device) => {
        let name = row.telegramFirstName ? `${row.telegramFirstName} ${row.telegramLastName} (${row.telegramPhone})` : '-';

        return <Flex gap={5} style={{ position: "relative" }}>
          <Typography.Text disabled={!!refreshingProfile[row.id]} >{name}</Typography.Text>
          {!!refreshingProfile[row.id] && (

            <SyncOutlined
              spin={!!refreshingProfile[row.id]}
              style={{
                color: refreshingProfile[row.id] ? '#1890ff' : '#999',
              }}
            />)}
        </Flex>;
      }

    },
    { title: 'Api Key', dataIndex: 'apiKey', key: 'apiKey' },
    {
      title: 'Status',
      key: 'status',
      render: (_: unknown, row: Device) => {
        const status = row.hasSession;
        const cfg = status ? statusConfig.has_session : statusConfig.no_session;
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
              key: 'test-api',
              icon: <ApiOutlined />,
              label: (
                <Link href={`/devices/${row.id}/test-api`}>
                  Test Api
                </Link>
              ),
            },
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

              disabled: !!refreshingProfile[row.id],
              key: 'refresh',
              icon: <SyncOutlined spin={!!refreshingProfile[row.id]} />,
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
          onClick: () => setDeleteTarget(row),
          label: (<> Hapus</>),
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

            <Popconfirm
              open={deleteTarget?.id === row.id}
              title="Hapus device ini?"
              onConfirm={() => {
                router.delete(`/devices/${row.id}`);
                setDeleteTarget(null);
              }}
              onCancel={() => setDeleteTarget(null)}
            >
              {/* dummy anchor supaya Popconfirm attach ke row context */}
              <span />
            </Popconfirm>
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