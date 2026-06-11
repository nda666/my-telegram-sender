import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  Typography,
} from 'antd';

import {
  LockOutlined,
  UserOutlined,
} from '@ant-design/icons';
import {
  useForm,
  usePage,
} from '@inertiajs/react';

type PageProps = {
  error?: string;
};

export default function Login() {
  const { error } = usePage<PageProps>().props;

  const { data, setData, post, processing } = useForm({
    username: '',
    password: '',
  });

  const handleSubmit = () => {
    post('/login');
  };

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#f0f2f5',
      }}
    >
      <Card style={{ width: 400 }}>
        <Typography.Title level={3} style={{ textAlign: 'center', marginBottom: 24 }}>
          Login
        </Typography.Title>

        {error && (
          <Alert title={error} type="error" showIcon style={{ marginBottom: 16 }} />
        )}

        <Form layout="vertical" onFinish={handleSubmit}>
          <Form.Item label="Username" required>
            <Input
              value={data.username}
              onChange={(e) => setData('username', e.target.value)}
              prefix={<UserOutlined />}
              placeholder="admin"
              size="large"
            />
          </Form.Item>

          <Form.Item label="Password" required>
            <Input.Password
              value={data.password}
              onChange={(e) => setData('password', e.target.value)}
              prefix={<LockOutlined />}
              placeholder="Password"
              size="large"
            />
          </Form.Item>

          <Button
            type="primary"
            htmlType="submit"
            block
            size="large"
            loading={processing}
          >
            Masuk
          </Button>
        </Form>
      </Card>
    </div>
  );
}