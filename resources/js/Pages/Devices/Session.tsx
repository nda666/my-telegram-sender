import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  Space,
  Steps,
  Typography,
} from 'antd';

import { ArrowLeftOutlined } from '@ant-design/icons';
import {
  Link,
  router,
  useForm,
  usePage,
} from '@inertiajs/react';

import AppLayout from '../../Layouts/AppLayout';

type Device = {
  id: number;
  name: string;
  phone: string;
  status: string;
  hasSession: boolean;
  telegramFirstName: string;
  telegramLastName: string;
  telegramPhone: string;
};

type PageProps = {
  device: Device;
  step: 'phone' | 'code' | 'password';
  phone?: string;
  error?: string;
  error_bag?: Record<string, string>;
};

export default function DeviceSession() {
  const { device, step, phone, error, error_bag } = usePage<PageProps>().props;
  const stepIndex = step === 'phone' ? 0 : step === 'code' ? 1 : 2;

  // Form untuk step code
  const codeForm = useForm({ code: '' });
  const submitCode = () => {
    codeForm.post(`/devices/${device.id}/session/code`, { preserveScroll: true });
  };

  // Form untuk step password
  const passwordForm = useForm({ password: '' });
  const submitPassword = () => {
    passwordForm.post(`/devices/${device.id}/session/password`, { preserveScroll: true });
  };

  // Submit step phone (trigger SendCode)
  const submitPhone = () => {
    router.post(`/devices/${device.id}/session`, {}, { preserveScroll: true });
  };

  return (
    <AppLayout>
      <Space orientation="vertical" size="large" style={{ width: '100%', maxWidth: 560 }}>
        <Link href="/devices">
          <Button type="link" icon={<ArrowLeftOutlined />} style={{ padding: 0 }}>
            Kembali
          </Button>
        </Link>

        <Typography.Title level={4} style={{ margin: 0 }}>
          Session Telegram — {device.name}
        </Typography.Title>

        {device.hasSession && (
          <Alert
            type="info"
            showIcon
            title={`Session aktif: ${device.telegramFirstName} (${device.telegramPhone})`}
            description="Login ulang akan menimpa session yang ada."
          />
        )}

        <Card>
          <Steps
            current={stepIndex}
            style={{ marginBottom: 24 }}
            items={[
              { title: 'Nomor' },
              { title: 'OTP' },
              { title: '2FA' },
            ]}
          />

          {error && (
            <Alert description={error} type="error" showIcon style={{ marginBottom: 16 }} />
          )}

          {/* STEP: PHONE */}
          {step === 'phone' && (
            <Form layout="vertical">
              <Typography.Paragraph style={{ fontSize: 16, marginBottom: 24 }}>
                Nomor HP Telegram: <strong>{device.phone}</strong>
              </Typography.Paragraph>
              <Space orientation="vertical" style={{ width: '100%' }}>
                <Button type="primary" block size="large" onClick={submitPhone}>
                  Kirim Kode OTP
                </Button>
                {/* Hanya tampil link ini kalau ada pending session (cookie ada di browser).
                    Backend tetap redirect ke /session kalau cookie tidak valid. */}
                <Link href={`/devices/${device.id}/session/code`}>
                  <Button type="link" block>
                    Saya sudah punya kode OTP
                  </Button>
                </Link>
              </Space>
            </Form>
          )}

          {/* STEP: CODE */}
          {step === 'code' && (
            <Form layout="vertical" onFinish={submitCode}>
              <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
                Kode dikirim ke <strong>{phone}</strong>
              </Typography.Paragraph>
              <Form.Item
                label="Kode OTP"
                required
                validateStatus={codeForm.errors.code || error_bag?.code ? 'error' : ''}
                help={codeForm.errors.code || error_bag?.code}
              >
                <Input
                  value={codeForm.data.code}
                  onChange={(e) => codeForm.setData('code', e.target.value)}
                  inputMode="numeric"
                  maxLength={6}
                  size="large"
                  disabled={codeForm.processing}
                  autoFocus
                />
              </Form.Item>
              <Space style={{ width: '100%', justifyContent: 'space-between' }}>
                <Link href={`/devices/${device.id}/session`}>
                  <Button type="link" style={{ padding: 0 }}>
                    ← Mulai ulang
                  </Button>
                </Link>
                <Button
                  type="primary"
                  htmlType="submit"
                  size="large"
                  loading={codeForm.processing}
                  disabled={codeForm.processing}
                >
                  Verifikasi
                </Button>
              </Space>
            </Form>
          )}

          {/* STEP: PASSWORD (2FA) */}
          {step === 'password' && (
            <Form layout="vertical" onFinish={submitPassword}>
              <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
                Akun ini mengaktifkan verifikasi dua langkah (2FA). Masukkan cloud password Telegram kamu.
              </Typography.Paragraph>
              <Form.Item
                label="Password 2FA"
                required
                validateStatus={passwordForm.errors.password ? 'error' : ''}
                help={passwordForm.errors.password}
              >
                <Input.Password
                  value={passwordForm.data.password}
                  onChange={(e) => passwordForm.setData('password', e.target.value)}
                  size="large"
                  disabled={passwordForm.processing}
                  autoFocus
                />
              </Form.Item>
              <Button
                type="primary"
                htmlType="submit"
                block
                size="large"
                loading={passwordForm.processing}
                disabled={passwordForm.processing}
              >
                Login
              </Button>
            </Form>
          )}
        </Card>
      </Space>
    </AppLayout>
  );
}