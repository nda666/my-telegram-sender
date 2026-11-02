import {
  Button,
  Flex,
  message as antMessage,
  Pagination,
  Table,
  Tag,
  Typography,
} from 'antd';

import {
  CopyOutlined,
  GlobalOutlined,
} from '@ant-design/icons';
import {
  Link,
  usePage,
} from '@inertiajs/react';

import AppLayout from '../../Layouts/AppLayout';

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
  logs: LogEntry[];
  pagination: { page: number; total: number; pages: number };
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

export default function LogsIndex() {
  const { logs, pagination } = usePage<PageProps>().props;

  const columns = [
    {
      title: 'Waktu',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
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
      render: (level: string) => (
        <Tag color={levelColor[level] ?? 'default'} style={{ textTransform: 'uppercase', fontWeight: 600, fontSize: 11 }}>
          {level}
        </Tag>
      ),
    },
    {
      title: 'Action',
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
      title: 'Device',
      dataIndex: 'deviceId',
      key: 'deviceId',
      width: 90,
      render: (id: number | null) =>
        id ? (
          <Link href={`/devices/${id}/logs`}>
            <Tag color="processing">#{id}</Tag>
          </Link>
        ) : (
          <span style={{ color: '#cbd5e1' }}>-</span>
        ),
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
      title: 'Pesan',
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
      <Typography.Title level={4}>System Logs</Typography.Title>
      <Table
        rowKey="id"
        dataSource={logs}
        columns={columns}
        pagination={false}
        size="small"
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
                  Tidak ada data payload tambahan
                </Typography.Text>
              )}
            </div>
          ),
          rowExpandable: (record) => Boolean(record.data),
        }}
      />
      {pagination.pages > 1 && (
        <div style={{ marginTop: 16, textAlign: 'right' }}>
          <Pagination
            current={pagination.page}
            total={pagination.total}
            pageSize={20}
            showSizeChanger={false}
            itemRender={(page, type, original) => {
              if (type === 'page') {
                return <Link href={`/logs?page=${page}`}>{page}</Link>;
              }
              return original;
            }}
          />
        </div>
      )}
    </AppLayout>
  );
}
