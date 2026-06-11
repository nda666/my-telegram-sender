import {
  useEffect,
  useState,
} from 'react';

import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  Space,
  Typography,
} from 'antd';

import { ArrowLeftOutlined } from '@ant-design/icons';
import {
  Link,
  useForm,
  usePage,
} from '@inertiajs/react';

import AppLayout from '../../Layouts/AppLayout';

type Device = {
  id: number;
  name: string;
  phone: string;
};

type PageProps = {
  device: Device | null;
  error?: string;
  error_bag?: Record<string, string>;
};

export default function DeviceForm() {
  const { device, error: serverError, error_bag: serverErrorBag } = usePage<PageProps>().props;
  const isEdit = !!device;

  // State lokal untuk menghandle "clear error" secara instan di UI saat klik submit
  const [localError, setLocalError] = useState<string | undefined>(serverError);
  const [localErrorBag, setLocalErrorBag] = useState<Record<string, string> | undefined>(serverErrorBag);

  // Sinkronisasi state lokal jika ada data error baru masuk dari server
  useEffect(() => {
    setLocalError(serverError);
    setLocalErrorBag(serverErrorBag);
  }, [serverError, serverErrorBag]);

  const { data, setData, post, put, processing, clearErrors } = useForm({
    name: device?.name ?? '',
    phone: device?.phone ?? '',
  });

  const submit = () => {
    // 1. Bersihkan errors bawaan useForm jika ada
    clearErrors();

    // 2. Bersihkan error visual (server props) secara instan agar user tahu proses baru dimulai
    setLocalError(undefined);
    setLocalErrorBag(undefined);

    const options = {
      preserveScroll: true,
      // Jika terjadi error pasca-request, state lokal otomatis diupdate lewat useEffect di atas
    };

    if (isEdit && device?.id) {
      put(`/devices/${device.id}`, options);
    } else {
      post('/devices', options);
    }
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
          {isEdit ? 'Edit Device' : 'Tambah Device'}
        </Typography.Title>

        <Card>
          {/* Menggunakan localError yang bisa di-clear instan */}
          {localError && (
            <Alert title={localError} type="error" showIcon style={{ marginBottom: 16 }} />
          )}

          <Form layout="vertical" onFinish={submit}>
            {/* Menggunakan localErrorBag untuk validasi per field */}
            <Form.Item
              label="Nama Device"
              required
              validateStatus={localErrorBag?.name ? 'error' : ''}
              help={localErrorBag?.name}
            >
              <Input
                value={data.name}
                onChange={(e) => setData('name', e.target.value)}
                placeholder="Device Jakarta 1"
                disabled={processing} // Disable input saat loading
              />
            </Form.Item>

            <Form.Item
              label="Nomor HP"
              required
              validateStatus={localErrorBag?.phone ? 'error' : ''}
              help={localErrorBag?.phone}
            >
              <Input
                value={data.phone}
                onChange={(e) => setData('phone', e.target.value)}
                placeholder="+628xxxxxxxxxx"
                disabled={processing} // Disable input saat loading
              />
            </Form.Item>



            {/* Button otomatis loading & disabled berkat 'processing' */}
            <Button type="primary" htmlType="submit" loading={processing} disabled={processing}>
              {isEdit ? 'Simpan' : 'Tambah'}
            </Button>
          </Form>
        </Card>
      </Space>
    </AppLayout>
  );
}