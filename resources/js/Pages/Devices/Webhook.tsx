import { useState } from 'react';

import {
  Alert,
  Avatar,
  Badge,
  Button,
  Card,
  Divider,
  Flex,
  Form,
  Input,
  message as antMessage,
  Space,
  Switch,
  Tabs,
  Tag,
  Typography,
} from 'antd';

import {
  ApiOutlined,
  ArrowLeftOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  CopyOutlined,
  GlobalOutlined,
  KeyOutlined,
  LinkOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
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
  hasSession: boolean;
  telegramFirstName?: string;
  telegramLastName?: string;
  telegramPhone?: string;
  avatarColor?: string;
  apiKey?: string;
  webhookUrl?: string;
  webhookSecret?: string;
  webhookEnabled?: boolean;
};

type PageProps = {
  device: Device;
  error?: string;
};

type TestResult = {
  success: boolean;
  statusCode: number;
  durationMs: number;
  responseBody?: string;
  error?: string;
} | null;

export default function DeviceWebhook() {
  const { device, error: propError } = usePage<PageProps>().props;

  const { data, setData, post, processing, errors, clearErrors } = useForm({
    webhookUrl: device.webhookUrl || '',
    webhookSecret: device.webhookSecret || '',
    webhookEnabled: device.webhookEnabled ?? true,
  });

  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<TestResult>(null);

  const handleSubmit = () => {
    clearErrors();
    post(`/devices/${device.id}/webhook`, {
      preserveScroll: true,
      only: ['device'],
      onSuccess: () => {
        antMessage.success('Konfigurasi Webhook berhasil disimpan');
      },
    });
  };

  const handleTestWebhook = async () => {
    if (!data.webhookUrl) {
      antMessage.warning('Silakan isi Webhook URL terlebih dahulu');
      return;
    }

    setTesting(true);
    setTestResult(null);

    try {
      const res = await fetch(`/devices/${device.id}/webhook/test`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest',
        },
        body: JSON.stringify({
          webhookUrl: data.webhookUrl,
          webhookSecret: data.webhookSecret,
        }),
      });

      const resultData: TestResult = await res.json();
      setTestResult(resultData);

      if (resultData && resultData.success) {
        antMessage.success(`Webhook test berhasil! (HTTP ${resultData.statusCode})`);
      } else {
        antMessage.error(`Webhook test gagal: ${resultData?.error || `HTTP ${resultData?.statusCode}`}`);
      }
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : 'Gagal menghubungi server';
      setTestResult({
        success: false,
        statusCode: 0,
        durationMs: 0,
        error: errMsg,
      });
      antMessage.error(errMsg);
    } finally {
      setTesting(false);
    }
  };

  const samplePayload = {
    event: 'message.received',
    deviceId: device.id,
    messageId: 10452,
    date: 1718452345,
    dateTime: '2026-06-15 14:32:25',
    text: 'Halo, saya ingin bertanya mengenai layanan ini...',
    out: false,
    sender: {
      id: 987654321,
      firstName: 'Budi',
      lastName: 'Santoso',
      username: 'budisantoso',
      phone: '+628123456789',
    },
    chat: {
      id: 987654321,
      type: 'user',
      title: 'Budi Santoso',
      username: 'budisantoso',
    },
    media: {
      type: 'photo',
      id: 543219876,
      fileName: '',
      mimeType: 'image/jpeg',
    },
  };

  const sampleHeaders = `POST /your-webhook-endpoint HTTP/1.1
Host: your-domain.com
Content-Type: application/json
User-Agent: Telegram-Sender-Webhook/1.0
X-Telegram-Sender-Event: message.received
X-Telegram-Sender-Device-Id: ${device.id}
${data.webhookSecret ? `X-Telegram-Sender-Signature: sha256=a1b2c3d4... (HMAC-SHA256)` : ''}`;

  const sampleLaravel = `<?php

namespace App\\Http\\Controllers;

use Illuminate\\Http\\Request;
use Illuminate\\Support\\Facades\\Log;

class TelegramWebhookController extends Controller
{
    /**
     * Handle incoming webhook message from Telegram Sender.
     * Route: Route::post('/api/telegram-webhook', [TelegramWebhookController::class, 'handle']);
     */
    public function handle(Request $request)
    {
        // 1. Verifikasi Signature HMAC-SHA256 (jika Secret Token diisi)
        $secret = config('services.telegram_sender.webhook_secret', '${data.webhookSecret || 'your-secret-key'}');
        $signature = $request->header('X-Telegram-Sender-Signature');

        if ($signature && $secret) {
            $expectedSignature = 'sha256=' . hash_hmac('sha256', $request->getContent(), $secret);
            if (!hash_equals($expectedSignature, $signature)) {
                return response()->json(['error' => 'Invalid signature'], 401);
            }
        }

        // 2. Ambil data pesan masuk
        $event     = $request->input('event');      // "message.received"
        $deviceId  = $request->input('deviceId');   // ID device di Telegram Sender
        $messageId = $request->input('messageId');  // ID pesan Telegram
        $text      = $request->input('text');       // Isi pesan teks
        $sender    = $request->input('sender');     // ['id', 'firstName', 'username', 'phone']
        $chat      = $request->input('chat');       // ['id', 'type', 'title']
        $media     = $request->input('media');      // ['type', 'id', 'fileName', 'mimeType']

        Log::info("Webhook Telegram [Device #{$deviceId}] dari {$sender['firstName']}: {$text}");

        // 3. Proses logika bisnis Anda (misal: bot otomatis, simpan ke DB)
        // ...

        return response()->json([
            'status' => 'success',
            'message' => 'Webhook received successfully'
        ], 200);
    }
}`;

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    antMessage.success(`${label} berhasil disalin`);
  };

  return (
    <AppLayout>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        {/* Top bar */}
        <Flex justify="space-between" align="center" wrap="wrap" gap={12}>
          <Space align="center">
            <Link href="/devices">
              <Button type="link" icon={<ArrowLeftOutlined />} style={{ padding: 0 }}>
                Kembali
              </Button>
            </Link>
            <Typography.Title level={4} style={{ margin: 0 }}>
              Webhook Pesan Masuk — {device.name}
            </Typography.Title>
          </Space>
          <Space>
            {device.hasSession ? (
              <Tag color="green" icon={<CheckCircleOutlined />}>
                Session Terhubung
              </Tag>
            ) : (
              <Tag color="default" icon={<CloseCircleOutlined />}>
                Belum Ada Session
              </Tag>
            )}
            {data.webhookEnabled && data.webhookUrl ? (
              <Tag color="processing" icon={<GlobalOutlined />}>
                Webhook Aktif
              </Tag>
            ) : (
              <Tag color="warning">Webhook Nonaktif</Tag>
            )}
          </Space>
        </Flex>

        {propError && (
          <Alert message={propError} type="error" showIcon style={{ marginBottom: 8 }} />
        )}

        {!device.hasSession && (
          <Alert
            type="warning"
            showIcon
            message="Device belum memiliki Session Telegram"
            description={
              <span>
                Agar webhook dapat menerima pesan secara otomatis, silakan buat session Telegram terlebih dahulu di menu{' '}
                <Link href={`/devices/${device.id}/session`}><strong>Session</strong></Link>.
              </span>
            }
          />
        )}

        {/* Main Grid */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))', gap: 24, alignItems: 'start' }}>
          {/* Left Column: Form & Test Panel */}
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            {/* Form Card */}
            <Card
              title={
                <Space>
                  <GlobalOutlined style={{ color: '#1890ff' }} />
                  <span>Pengaturan Webhook</span>
                </Space>
              }
              bordered={false}
              style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}
            >
              <Form layout="vertical" onFinish={handleSubmit}>
                <Form.Item label="Status Webhook">
                  <Flex align="center" gap={12}>
                    <Switch
                      checked={data.webhookEnabled}
                      onChange={(checked) => setData('webhookEnabled', checked)}
                    />
                    <Typography.Text type={data.webhookEnabled ? undefined : 'secondary'}>
                      {data.webhookEnabled ? 'Aktif (Meneruskan pesan masuk baru)' : 'Nonaktif (Pesan tidak diteruskan)'}
                    </Typography.Text>
                  </Flex>
                </Form.Item>

                <Form.Item
                  label="Webhook URL"
                  required
                  validateStatus={errors.webhookUrl ? 'error' : ''}
                  help={errors.webhookUrl || 'URL tujuan yang akan menerima HTTP POST saat ada pesan masuk baru'}
                >
                  <Input
                    prefix={<LinkOutlined style={{ color: '#bfbfbf' }} />}
                    placeholder="https://domain-anda.com/api/webhook/telegram"
                    value={data.webhookUrl}
                    onChange={(e) => setData('webhookUrl', e.target.value)}
                    size="large"
                  />
                </Form.Item>

                <Form.Item
                  label="Secret Token / Signature Secret (Opsional)"
                  help="Jika diisi, payload akan ditandatangani dengan HMAC-SHA256 di header X-Telegram-Sender-Signature"
                >
                  <Input.Password
                    prefix={<KeyOutlined style={{ color: '#bfbfbf' }} />}
                    placeholder="Rahasia / Signature Key"
                    value={data.webhookSecret}
                    onChange={(e) => setData('webhookSecret', e.target.value)}
                    size="large"
                  />
                </Form.Item>

                <Flex justify="space-between" align="center" style={{ marginTop: 24 }}>
                  <Button
                    type="primary"
                    htmlType="submit"
                    loading={processing}
                    size="large"
                    icon={<GlobalOutlined />}
                  >
                    Simpan Konfigurasi
                  </Button>
                  <Button
                    icon={<ThunderboltOutlined />}
                    onClick={handleTestWebhook}
                    loading={testing}
                    disabled={!data.webhookUrl}
                    size="large"
                  >
                    Test Webhook
                  </Button>
                </Flex>
              </Form>
            </Card>

            {/* Test Result Card */}
            {testResult && (
              <Card
                title={
                  <Space>
                    <ApiOutlined style={{ color: testResult.success ? '#52c41a' : '#ff4d4f' }} />
                    <span>Hasil Tes Webhook</span>
                  </Space>
                }
                extra={
                  <Space>
                    <Tag color={testResult.success ? 'success' : 'error'}>
                      {testResult.statusCode > 0 ? `HTTP ${testResult.statusCode}` : 'Error Koneksi'}
                    </Tag>
                    <Tag color="default">{testResult.durationMs} ms</Tag>
                  </Space>
                }
                bordered={false}
                style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}
              >
                {testResult.error && (
                  <Alert
                    message={testResult.error}
                    type="error"
                    showIcon
                    style={{ marginBottom: 12 }}
                  />
                )}

                <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 6 }}>
                  Response Body:
                </Typography.Text>
                <pre
                  style={{
                    background: testResult.success ? '#f6ffed' : '#fff2f0',
                    border: `1px solid ${testResult.success ? '#b7eb8f' : '#ffccc7'}`,
                    borderRadius: 6,
                    padding: 12,
                    fontSize: 12,
                    margin: 0,
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-all',
                    maxHeight: 180,
                    overflowY: 'auto',
                  }}
                >
                  {testResult.responseBody || '(Tidak ada response body)'}
                </pre>
              </Card>
            )}

            {/* Device Detail Card */}
            <Card
              bordered={false}
              style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}
            >
              <Flex align="center" gap={12}>
                <Badge status={device.hasSession ? 'success' : 'default'}>
                  <Avatar size={44} style={{ background: device.avatarColor || '#1890ff' }}>
                    {device.telegramFirstName?.charAt(0)?.toUpperCase() || device.name?.charAt(0)?.toUpperCase() || 'D'}
                  </Avatar>
                </Badge>
                <Flex vertical gap={0}>
                  <Typography.Text strong style={{ fontSize: 15 }}>
                    {device.telegramFirstName ? `${device.telegramFirstName} ${device.telegramLastName || ''}` : device.name}
                  </Typography.Text>
                  <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                    Nomor: {device.telegramPhone || device.phone}
                  </Typography.Text>
                </Flex>
                <Flex vertical align="flex-end" style={{ marginLeft: 'auto' }}>
                  <Typography.Text type="secondary" style={{ fontSize: 11 }}>
                    Device ID
                  </Typography.Text>
                  <Typography.Text code style={{ fontSize: 12 }}>
                    #{device.id}
                  </Typography.Text>
                </Flex>
              </Flex>
            </Card>
          </Space>

          {/* Right Column: Documentation & Sample Payload */}
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            <Card
              title={
                <Space>
                  <ApiOutlined style={{ color: '#52c41a' }} />
                  <span>Format Data Webhook</span>
                </Space>
              }
              bordered={false}
              style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}
            >
              <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
                Setiap kali ada pesan baru masuk ke akun Telegram device ini, sistem akan mengirimkan HTTP POST request ke Webhook URL dengan format berikut:
              </Typography.Paragraph>

              <Tabs
                defaultActiveKey="body"
                size="small"
                items={[
                  {
                    key: 'body',
                    label: 'Payload JSON',
                    children: (
                      <div>
                        <Flex justify="space-between" align="center" style={{ marginBottom: 8 }}>
                          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                            Body (application/json)
                          </Typography.Text>
                          <Button
                            size="small"
                            icon={<CopyOutlined />}
                            onClick={() => copyToClipboard(JSON.stringify(samplePayload, null, 2), 'Sample Payload JSON')}
                          >
                            Salin JSON
                          </Button>
                        </Flex>
                        <pre
                          style={{
                            background: '#1f1f1f',
                            color: '#a6e22e',
                            padding: 16,
                            borderRadius: 8,
                            fontSize: 12,
                            maxHeight: 340,
                            overflowY: 'auto',
                            margin: 0,
                          }}
                        >
                          {JSON.stringify(samplePayload, null, 2)}
                        </pre>
                      </div>
                    ),
                  },
                  {
                    key: 'headers',
                    label: 'HTTP Headers',
                    children: (
                      <div>
                        <Flex justify="space-between" align="center" style={{ marginBottom: 8 }}>
                          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                            Request Headers
                          </Typography.Text>
                          <Button
                            size="small"
                            icon={<CopyOutlined />}
                            onClick={() => copyToClipboard(sampleHeaders, 'HTTP Headers')}
                          >
                            Salin Headers
                          </Button>
                        </Flex>
                        <pre
                          style={{
                            background: '#1f1f1f',
                            color: '#66d9ef',
                            padding: 16,
                            borderRadius: 8,
                            fontSize: 12,
                            maxHeight: 340,
                            overflowY: 'auto',
                            margin: 0,
                          }}
                        >
                          {sampleHeaders}
                        </pre>
                      </div>
                    ),
                  },
                  {
                    key: 'laravel',
                    label: 'PHP Laravel',
                    children: (
                      <div>
                        <Flex justify="space-between" align="center" style={{ marginBottom: 8 }}>
                          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                            Contoh Controller Handler (Laravel)
                          </Typography.Text>
                          <Button
                            size="small"
                            icon={<CopyOutlined />}
                            onClick={() => copyToClipboard(sampleLaravel, 'Sample Code Laravel')}
                          >
                            Salin Code
                          </Button>
                        </Flex>
                        <pre
                          style={{
                            background: '#1f1f1f',
                            color: '#e6db74',
                            padding: 16,
                            borderRadius: 8,
                            fontSize: 12,
                            maxHeight: 340,
                            overflowY: 'auto',
                            margin: 0,
                          }}
                        >
                          {sampleLaravel}
                        </pre>
                      </div>
                    ),
                  },
                ]}
              />

              <Divider style={{ margin: '16px 0' }} />

              <Typography.Title level={5} style={{ fontSize: 13, marginBottom: 8 }}>
                Penjelasan Field:
              </Typography.Title>
              <ul style={{ paddingLeft: 18, margin: 0, fontSize: 12, color: '#595959', lineHeight: '1.8' }}>
                <li><code>event</code>: Tipe event, contoh <code>message.received</code>.</li>
                <li><code>deviceId</code>: ID device pengirim/penerima di sistem ini.</li>
                <li><code>messageId</code>: ID pesan Telegram.</li>
                <li><code>text</code>: Teks atau pesan yang dikirimkan.</li>
                <li><code>sender</code>: Detail pengirim (ID Telegram, Nama, Username, Nomor HP).</li>
                <li><code>chat</code>: Tipe obrolan (<code>user</code> untuk personal, <code>chat</code> untuk grup, <code>channel</code>).</li>
                <li><code>media</code>: Informasi lampiran media jika pesan memuat foto, dokumen, video, audio, atau voice.</li>
              </ul>
            </Card>
          </Space>
        </div>
      </Space>
    </AppLayout>
  );
}
