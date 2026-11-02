import { useState } from 'react';

import {
  Avatar,
  Badge,
  Button,
  Card,
  Flex,
  Input,
  message as antMessage,
  Pagination,
  Radio,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';

import {
  ArrowLeftOutlined,
  CopyOutlined,
  FileTextOutlined,
  GlobalOutlined,
  ReloadOutlined,
  RobotOutlined,
  SearchOutlined,
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
  hasSession: boolean;
  hasBot: boolean;
  botUsername?: string;
  botFirstName?: string;
  botEnabled?: boolean;
  telegramFirstName?: string;
  telegramLastName?: string;
  telegramPhone?: string;
  avatarColor?: string;
  apiKey?: string;
};

type LogEntry = {
  id: number;
  deviceId: number | null;
  level: string;
  action: string;
  message: string;
  ip?: string;
  data?: string;
  createdAt: string;
};

type PageProps = {
  device: Device;
  logs: LogEntry[];
  pagination: {
    page: number;
    total: number;
    pages: number;
  };
};

const levelColor: Record<string, string> = {
  info: 'blue',
  error: 'red',
  warn: 'orange',
  warning: 'orange',
};

function formatJSON(str?: string): string {
  if (!str) return '';
  try {
    return JSON.stringify(JSON.parse(str), null, 2);
  } catch {
    return str;
  }
}

export default function DeviceLogs() {
  const { device, logs, pagination } = usePage<PageProps>().props;
  const [levelFilter, setLevelFilter] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [isRefreshing, setIsRefreshing] = useState<boolean>(false);

  const handleRefresh = () => {
    setIsRefreshing(true);
    router.reload({
      onFinish: () => setIsRefreshing(false),
    });
  };

  const filteredLogs = logs.filter(log => {
    if (levelFilter !== 'all' && log.level.toLowerCase() !== levelFilter.toLowerCase()) {
      return false;
    }
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      const matchMsg = log.message.toLowerCase().includes(q);
      const matchAct = log.action.toLowerCase().includes(q);
      const matchIP = (log.ip || '').toLowerCase().includes(q);
      const matchData = (log.data || '').toLowerCase().includes(q);
      if (!matchMsg && !matchAct && !matchIP && !matchData) {
        return false;
      }
    }
    return true;
  });

  const columns = [
    {
      title: 'Waktu',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 165,
      render: (val: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12, color: '#64748b' }}>
          {val}
        </span>
      ),
    },
    {
      title: 'Level',
      dataIndex: 'level',
      key: 'level',
      width: 85,
      render: (level: string) => {
        const lvl = level.toLowerCase();
        return (
          <Tag color={levelColor[lvl] ?? 'default'} style={{ textTransform: 'uppercase', fontWeight: 600, fontSize: 11 }}>
            {level}
          </Tag>
        );
      },
    },
    {
      title: 'Aksi / Event',
      dataIndex: 'action',
      key: 'action',
      width: 140,
      render: (action: string) => {
        let tagColor = 'geekblue';
        if (action.startsWith('api.')) tagColor = 'purple';
        else if (action.startsWith('webhook.')) tagColor = 'cyan';
        else if (action.startsWith('bot.')) tagColor = 'magenta';
        else if (action.startsWith('session.')) tagColor = 'gold';

        return (
          <Tag color={tagColor} style={{ fontFamily: 'monospace', fontSize: 12 }}>
            {action}
          </Tag>
        );
      },
    },
    {
      title: 'IP Address',
      dataIndex: 'ip',
      key: 'ip',
      width: 135,
      render: (ip: string) => {
        if (!ip) return <span style={{ color: '#cbd5e1' }}>-</span>;
        return (
          <Tag icon={<GlobalOutlined />} style={{ fontFamily: 'monospace', fontSize: 12 }}>
            {ip}
          </Tag>
        );
      },
    },
    {
      title: 'Pesan Log',
      dataIndex: 'message',
      key: 'message',
      render: (msg: string) => (
        <Typography.Paragraph
          style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontSize: 13 }}
          ellipsis={{ rows: 3, expandable: true, symbol: 'Lihat selengkapnya' }}
        >
          {msg}
        </Typography.Paragraph>
      ),
    },
  ];

  return (
    <AppLayout>
      {/* ── HEADER INFORMASI DEVICE ── */}
      <Card style={{ marginBottom: 16 }}>
        <Flex justify="space-between" align="center" wrap="wrap" gap={12}>
          <Flex align="center" gap={12}>
            <Link href="/devices">
              <Button icon={<ArrowLeftOutlined />} shape="circle" />
            </Link>
            <Avatar
              size={48}
              style={{
                backgroundColor: device.avatarColor || '#3b82f6',
                fontWeight: 600,
                fontSize: 18,
              }}
            >
              {device.name.slice(0, 2).toUpperCase()}
            </Avatar>
            <div>
              <Flex align="center" gap={8}>
                <Typography.Title level={4} style={{ margin: 0 }}>
                  {device.name}
                </Typography.Title>
                {device.hasSession ? (
                  <Badge status="success" text="User MTProto Aktif" />
                ) : (
                  <Badge status="default" text="No User Session" />
                )}
                {device.hasBot && (
                  <Tag color={device.botEnabled ? 'cyan' : 'default'} icon={<RobotOutlined />}>
                    @{device.botUsername || 'bot'} ({device.botEnabled ? 'Aktif' : 'Nonaktif'})
                  </Tag>
                )}
              </Flex>
              <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                Device #{device.id} • {device.phone || 'Tanpa nomor'}
              </Typography.Text>
            </div>
          </Flex>

          <Space>
            <Tooltip title="Muat ulang log terbaru">
              <Button
                icon={<ReloadOutlined spin={isRefreshing} />}
                onClick={handleRefresh}
              >
                Refresh
              </Button>
            </Tooltip>
          </Space>
        </Flex>
      </Card>

      {/* ── LOGS TABLE & FILTERS ── */}
      <Card
        title={
          <Flex align="center" gap={8}>
            <FileTextOutlined style={{ color: '#1677ff' }} />
            <span>Riwayat Log Device</span>
            <Tag color="blue">{pagination.total} Total</Tag>
          </Flex>
        }
        extra={
          <Flex align="center" gap={12} wrap="wrap">
            <Radio.Group
              size="small"
              value={levelFilter}
              onChange={e => setLevelFilter(e.target.value)}
              buttonStyle="solid"
            >
              <Radio.Button value="all">Semua</Radio.Button>
              <Radio.Button value="info">Info</Radio.Button>
              <Radio.Button value="warn">Warn</Radio.Button>
              <Radio.Button value="error">Error</Radio.Button>
            </Radio.Group>

            <Input
              size="small"
              placeholder="Cari aksi, pesan, IP, payload..."
              prefix={<SearchOutlined style={{ color: '#94a3b8' }} />}
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              allowClear
              style={{ width: 230 }}
            />
          </Flex>
        }
      >
        <Table
          rowKey="id"
          dataSource={filteredLogs}
          columns={columns}
          pagination={false}
          size="middle"
          locale={{ emptyText: 'Belum ada log untuk device ini' }}
          expandable={{
            expandedRowRender: (record) => (
              <div style={{ background: '#f8fafc', padding: '12px 16px', borderRadius: 8, border: '1px solid #e2e8f0' }}>
                <Flex justify="space-between" align="center" style={{ marginBottom: 8 }}>
                  <Typography.Text strong style={{ fontSize: 13 }}>
                    Detail Request Payload:
                  </Typography.Text>
                  {record.data && (
                    <Button
                      size="small"
                      icon={<CopyOutlined />}
                      onClick={() => {
                        navigator.clipboard.writeText(record.data || '');
                        antMessage.success('Data JSON berhasil disalin!');
                      }}
                    >
                      Salin JSON
                    </Button>
                  )}
                </Flex>

                {record.data ? (
                  <pre
                    style={{
                      background: '#0f172a',
                      color: '#38bdf8',
                      padding: '12px 16px',
                      borderRadius: 6,
                      fontSize: 12,
                      fontFamily: 'monospace',
                      overflowX: 'auto',
                      maxHeight: 280,
                      margin: 0,
                    }}
                  >
                    {formatJSON(record.data)}
                  </pre>
                ) : (
                  <Typography.Text type="secondary" italic style={{ fontSize: 12 }}>
                    Tidak ada payload tambahan untuk event ini.
                  </Typography.Text>
                )}
              </div>
            ),
            rowExpandable: (record) => Boolean(record.data),
          }}
        />

        {pagination.pages > 1 && (
          <Flex justify="flex-end" style={{ marginTop: 16 }}>
            <Pagination
              current={pagination.page}
              total={pagination.total}
              pageSize={20}
              showSizeChanger={false}
              itemRender={(page, type, original) => {
                if (type === 'page') {
                  return <Link href={`/devices/${device.id}/logs?page=${page}`}>{page}</Link>;
                }
                return original;
              }}
            />
          </Flex>
        )}
      </Card>
    </AppLayout>
  );
}
