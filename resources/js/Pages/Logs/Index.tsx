import { Link, usePage } from '@inertiajs/react';
import { Table, Tag, Typography, Pagination } from 'antd';
import AppLayout from '../../Layouts/AppLayout';

type LogEntry = {
  id: number;
  deviceId: number | null;
  level: string;
  action: string;
  message: string;
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
};

export default function LogsIndex() {
  const { logs, pagination } = usePage<PageProps>().props;

  const columns = [
    { title: 'Waktu', dataIndex: 'createdAt', key: 'createdAt', width: 180 },
    {
      title: 'Level',
      dataIndex: 'level',
      key: 'level',
      width: 90,
      render: (level: string) => (
        <Tag color={levelColor[level] ?? 'default'}>{level}</Tag>
      ),
    },
    { title: 'Action', dataIndex: 'action', key: 'action', width: 160 },
    {
      title: 'Device',
      dataIndex: 'deviceId',
      key: 'deviceId',
      width: 80,
      render: (id: number | null) => (id ? `#${id}` : '-'),
    },
    { title: 'Pesan', dataIndex: 'message', key: 'message' },
  ];

  return (
    <AppLayout>
      <Typography.Title level={4}>Logs</Typography.Title>
      <Table
        rowKey="id"
        dataSource={logs}
        columns={columns}
        pagination={false}
        size="small"
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
