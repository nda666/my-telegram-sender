import { useEffect, useState } from 'react';

import {
  Alert,
  Avatar,
  Badge,
  Button,
  Card,
  Divider,
  Empty,
  Flex,
  Form,
  Input,
  message as antMessage,
  Modal,
  Popconfirm,
  Radio,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';

import {
  ApiOutlined,
  AppstoreAddOutlined,
  ArrowLeftOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  CopyOutlined,
  DeleteOutlined,
  DisconnectOutlined,
  EditOutlined,
  GlobalOutlined,
  KeyOutlined,
  LinkOutlined,
  PhoneOutlined,
  PlusOutlined,
  ReloadOutlined,
  RobotOutlined,
  SaveOutlined,
  ThunderboltOutlined,
  UserOutlined,
} from '@ant-design/icons';
import {
  Link,
  router,
  useForm,
  usePage,
} from '@inertiajs/react';

import AppLayout from '../../Layouts/AppLayout';

type BotCommand = {
  command: string;
  description: string;
};

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
  hasBot?: boolean;
  botToken?: string;
  botUsername?: string;
  botFirstName?: string;
  botId?: number;
  botEnabled?: boolean;
  botCommands?: string;
  botMenuSource?: 'manual' | 'url';
  botMenuUrl?: string;
  botRequestPhoneOnStart?: boolean;
  botStartMessage?: string;
  botPhoneButtonText?: string;
  botPhoneReceivedMessage?: string;
  webhookUrl?: string;
};

type PageProps = {
  device: Device;
  error?: string;
};

export default function DeviceBot() {
  const { device, error: propError } = usePage<PageProps>().props;

  const { data, setData, post, processing, errors, clearErrors } = useForm({
    botToken: device.botToken || '',
    botEnabled: device.botEnabled ?? false,
  });

  const [disconnecting, setDisconnecting] = useState(false);
  const [toggling, setToggling] = useState(false);

  // Menu Source Config Form
  const {
    data: menuConfig,
    setData: setMenuConfig,
    post: postMenuConfig,
    processing: savingMenuConfig,
  } = useForm({
    botMenuSource: (device.botMenuSource || 'manual') as 'manual' | 'url',
    botMenuUrl: device.botMenuUrl || '',
  });

  const handleSaveMenuConfig = () => {
    postMenuConfig(`/devices/${device.id}/bot/menu-config`, {
      preserveScroll: true,
      only: ['device'],
      onSuccess: () => {
        antMessage.success('Pengaturan Sumber Menu Bot berhasil disimpan');
      },
    });
  };

  // Request Phone on Start Form
  const {
    data: startConfig,
    setData: setStartConfig,
    post: postStartConfig,
    processing: savingStartConfig,
  } = useForm({
    botRequestPhoneOnStart: device.botRequestPhoneOnStart ?? false,
    botStartMessage:
      device.botStartMessage ||
      '👋 Halo! Selamat datang. Silakan tekan tombol di bawah ini untuk membagikan nomor telepon Anda:',
    botPhoneButtonText: device.botPhoneButtonText || '📱 Bagikan Nomor Telepon',
    botPhoneReceivedMessage:
      device.botPhoneReceivedMessage ||
      '✅ Terima kasih! Nomor telepon Anda telah berhasil diterima.',
  });

  const handleSaveStartConfig = () => {
    postStartConfig(`/devices/${device.id}/bot/start-config`, {
      preserveScroll: true,
      only: ['device'],
      onSuccess: () => {
        antMessage.success('Pengaturan Request Phone saat /start berhasil disimpan');
      },
    });
  };

  // Bot Commands State
  const initialCommands: BotCommand[] = (() => {
    if (device.botCommands) {
      try {
        const parsed = JSON.parse(device.botCommands);
        if (Array.isArray(parsed)) return parsed;
      } catch {
        // ignore
      }
    }
    return [
      { command: 'start', description: 'Mulai bot dan tampilkan menu utama' },
      { command: 'saldo', description: 'Cek saldo dan poin akun Anda' },
      { command: 'bantuan', description: 'Hubungi layanan pelanggan / customer service' },
    ];
  })();

  const [commands, setCommands] = useState<BotCommand[]>(initialCommands);
  const [savingCommands, setSavingCommands] = useState(false);
  const [loadingCommands, setLoadingCommands] = useState(false);

  // Modal State
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [cmdForm] = Form.useForm();

  // Fetch live commands from Telegram on mount if bot is connected
  useEffect(() => {
    if (device.hasBot) {
      fetchCommandsFromTelegram();
    }
  }, [device.id, device.hasBot]);

  const fetchCommandsFromTelegram = async () => {
    setLoadingCommands(true);
    try {
      const res = await fetch(`/devices/${device.id}/bot/commands`);
      const resData = await res.json();
      if (resData.commands && Array.isArray(resData.commands)) {
        if (resData.commands.length > 0) {
          setCommands(resData.commands);
        }
      }
    } catch {
      // ignore
    } finally {
      setLoadingCommands(false);
    }
  };

  const handleConnect = () => {
    clearErrors();
    if (!data.botToken.trim()) {
      antMessage.warning('Silakan masukkan Bot Token terlebih dahulu');
      return;
    }

    post(`/devices/${device.id}/bot/connect`, {
      preserveScroll: true,
      only: ['device'],
      onSuccess: () => {
        antMessage.success('Telegram Bot berhasil terhubung ke MTProto');
      },
    });
  };

  const handleDisconnect = () => {
    setDisconnecting(true);
    post(`/devices/${device.id}/bot/disconnect`, {
      preserveScroll: true,
      only: ['device'],
      onSuccess: () => {
        antMessage.success('Telegram Bot berhasil diputuskan');
      },
      onFinish: () => {
        setDisconnecting(false);
      },
    });
  };

  const handleToggle = (checked: boolean) => {
    setToggling(true);
    setData('botEnabled', checked);
    router.post(
      `/devices/${device.id}/bot/toggle`,
      { botEnabled: checked },
      {
        preserveScroll: true,
        only: ['device'],
        onSuccess: () => {
          antMessage.success(checked ? 'Bot MTProto diaktifkan' : 'Bot MTProto dinonaktifkan');
        },
        onFinish: () => {
          setToggling(false);
        },
      }
    );
  };

  // Commands Handlers
  const handleOpenAddModal = () => {
    setEditingIndex(null);
    cmdForm.resetFields();
    setIsModalOpen(true);
  };

  const handleOpenEditModal = (index: number) => {
    setEditingIndex(index);
    cmdForm.setFieldsValue({
      command: commands[index].command,
      description: commands[index].description,
    });
    setIsModalOpen(true);
  };

  const handleSaveModal = () => {
    cmdForm.validateFields().then((values) => {
      const cleanCmd = values.command.trim().replace(/^\//, '').toLowerCase();
      const cleanDesc = values.description.trim();

      const newCommand: BotCommand = { command: cleanCmd, description: cleanDesc };

      if (editingIndex !== null) {
        const updated = [...commands];
        updated[editingIndex] = newCommand;
        setCommands(updated);
      } else {
        // Cek duplikasi
        if (commands.some((c) => c.command.toLowerCase() === cleanCmd)) {
          antMessage.warning(`Command /${cleanCmd} sudah ada dalam daftar`);
          return;
        }
        setCommands([...commands, newCommand]);
      }

      setIsModalOpen(false);
      antMessage.success('Menu perintah berhasil ditambahkan ke daftar.');
    });
  };

  const handleDeleteCommand = (index: number) => {
    const updated = commands.filter((_, i) => i !== index);
    setCommands(updated);
    antMessage.success('Menu perintah dihapus dari daftar.');
  };

  const handleApplyCommandsToTelegram = async () => {
    setSavingCommands(true);
    try {
      const res = await fetch(`/devices/${device.id}/bot/commands`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest',
        },
        body: JSON.stringify({ commands }),
      });

      const resData = await res.json();
      if (res.ok && resData.success) {
        antMessage.success(`Berhasil menerapkan ${commands.length} menu perintah ke server Telegram!`);
      } else {
        antMessage.error(`Gagal menerapkan menu: ${resData.error || 'Server error'}`);
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Gagal menghubungi server';
      antMessage.error(msg);
    } finally {
      setSavingCommands(false);
    }
  };

  const handleAddTemplate = (cmd: string, desc: string) => {
    const cleanCmd = cmd.trim().replace(/^\//, '').toLowerCase();
    if (commands.some((c) => c.command.toLowerCase() === cleanCmd)) {
      antMessage.info(`Command /${cleanCmd} sudah ada`);
      return;
    }
    setCommands([...commands, { command: cleanCmd, description: desc }]);
    antMessage.success(`Menu /${cleanCmd} ditambahkan`);
  };

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    antMessage.success(`${label} berhasil disalin`);
  };

  const sampleCallbackPayload = {
    event: 'callback_query',
    deviceId: device.id,
    queryId: 2847591029384,
    date: 1718452345,
    dateTime: '2026-06-15 14:32:25',
    data: 'cek_saldo',
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
    },
  };

  const sampleContactPayload = {
    event: 'message.received',
    deviceId: device.id,
    messageId: 10488,
    date: 1718452350,
    dateTime: '2026-06-15 14:32:30',
    text: '',
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
    },
    media: {
      type: 'contact',
      id: 987654321,
      fileName: 'Budi Santoso',
      mimeType: '',
    },
  };

  const commandColumns: ColumnsType<BotCommand> = [
    {
      title: 'Perintah (Command)',
      dataIndex: 'command',
      key: 'command',
      width: 170,
      render: (cmd: string) => (
        <Tag color="blue" style={{ fontFamily: 'monospace', fontSize: 13, padding: '2px 8px' }}>
          /{cmd}
        </Tag>
      ),
    },
    {
      title: 'Deskripsi Menu',
      dataIndex: 'description',
      key: 'description',
      render: (desc: string) => <span style={{ fontSize: 13 }}>{desc}</span>,
    },
    {
      title: 'Aksi',
      key: 'actions',
      width: 110,
      render: (_, __, index: number) => (
        <Space size="small">
          <Button
            type="text"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleOpenEditModal(index)}
          />
          <Popconfirm
            title="Hapus menu perintah ini?"
            onConfirm={() => handleDeleteCommand(index)}
            okText="Ya"
            cancelText="Batal"
          >
            <Button type="text" danger size="small" icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

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
              Telegram Bot (MTProto gotd) — {device.name}
            </Typography.Title>
          </Space>
          <Space>
            {device.hasSession ? (
              <Tag color="green" icon={<CheckCircleOutlined />}>
                Userbot Terhubung
              </Tag>
            ) : (
              <Tag color="default" icon={<CloseCircleOutlined />}>
                Userbot Belum Login
              </Tag>
            )}
            {device.hasBot ? (
              <Tag color="cyan" icon={<RobotOutlined />}>
                Bot: @{device.botUsername || 'connected'}
              </Tag>
            ) : (
              <Tag color="default">Bot Belum Terhubung</Tag>
            )}
            {device.hasBot && (
              <Tag color={device.botEnabled ? 'processing' : 'warning'}>
                {device.botEnabled ? 'Bot Aktif' : 'Bot Nonaktif'}
              </Tag>
            )}
          </Space>
        </Flex>

        {propError && (
          <Alert message={propError} type="error" showIcon style={{ marginBottom: 8 }} />
        )}

        {/* Main Grid */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(420px, 1fr))', gap: 24, alignItems: 'start' }}>
          {/* Left Column: Bot Status & Form */}
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            {/* Bot Profile / Status Card */}
            {device.hasBot ? (
              <Card
                title={
                  <Space>
                    <RobotOutlined style={{ color: '#13c2c2' }} />
                    <span>Profil Telegram Bot MTProto</span>
                  </Space>
                }
                extra={
                  <Popconfirm
                    title="Putuskan Telegram Bot?"
                    description="Sesi MTProto bot ini akan dihapus dari device."
                    onConfirm={handleDisconnect}
                    okText="Ya, Putuskan"
                    cancelText="Batal"
                    okButtonProps={{ danger: true }}
                  >
                    <Button
                      size="small"
                      danger
                      icon={<DisconnectOutlined />}
                      loading={disconnecting}
                    >
                      Putuskan Bot
                    </Button>
                  </Popconfirm>
                }
                bordered={false}
                style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}
              >
                <Flex align="center" gap={16} style={{ marginBottom: 20 }}>
                  <Avatar size={56} style={{ background: '#13c2c2', fontSize: 24 }}>
                    <RobotOutlined />
                  </Avatar>
                  <Flex vertical gap={2}>
                    <Typography.Text strong style={{ fontSize: 16 }}>
                      {device.botFirstName || 'Telegram Bot'}
                    </Typography.Text>
                    <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                      @{device.botUsername}
                    </Typography.Text>
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      Bot ID: <code>{device.botId}</code>
                    </Typography.Text>
                  </Flex>
                  <Tag color="success" style={{ marginLeft: 'auto', alignSelf: 'flex-start' }}>
                    MTProto Online
                  </Tag>
                </Flex>

                <Divider style={{ margin: '12px 0' }} />

                <Flex justify="space-between" align="center">
                  <div>
                    <Typography.Text strong style={{ display: 'block', fontSize: 13 }}>
                      Status Aktif Bot
                    </Typography.Text>
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      {device.botEnabled
                        ? 'Bot menerima pesan & callback query interaktif'
                        : 'Bot sementara dinonaktifkan'}
                    </Typography.Text>
                  </div>
                  <Switch
                    checked={device.botEnabled}
                    loading={toggling}
                    onChange={handleToggle}
                  />
                </Flex>
              </Card>
            ) : (
              <Card
                title={
                  <Space>
                    <RobotOutlined style={{ color: '#1890ff' }} />
                    <span>Hubungkan Telegram Bot Token</span>
                  </Space>
                }
                bordered={false}
                style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}
              >
                <Typography.Paragraph type="secondary" style={{ fontSize: 13, marginBottom: 20 }}>
                  Masukkan <strong>Bot Token</strong> dari <code>@BotFather</code>. Sistem akan mengautentikasi bot ke MTProto Telegram secara otomatis menggunakan library <code>gotd/td</code>.
                </Typography.Paragraph>

                <Form layout="vertical" onFinish={handleConnect}>
                  <Form.Item
                    label="Bot Token"
                    required
                    validateStatus={errors.botToken ? 'error' : ''}
                    help={errors.botToken || 'Format: 123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ'}
                  >
                    <Input.Password
                      prefix={<KeyOutlined style={{ color: '#bfbfbf' }} />}
                      placeholder="123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ"
                      value={data.botToken}
                      onChange={(e) => setData('botToken', e.target.value)}
                      size="large"
                    />
                  </Form.Item>

                  <Button
                    type="primary"
                    htmlType="submit"
                    loading={processing}
                    size="large"
                    icon={<RobotOutlined />}
                    style={{ width: '100%', marginTop: 8 }}
                  >
                    Hubungkan Bot via MTProto
                  </Button>
                </Form>
              </Card>
            )}

            {/* Otomatis Request Phone Number saat /start Card */}
            {device.hasBot && (
              <Card
                title={
                  <Space>
                    <PhoneOutlined style={{ color: '#52c41a' }} />
                    <span>Otomatis Minta Nomor HP saat /start (Request Phone)</span>
                  </Space>
                }
                bordered={false}
                style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}
              >
                <Typography.Paragraph type="secondary" style={{ fontSize: 12, marginBottom: 16 }}>
                  Saat pengguna membuka atau mengirimkan <code>/start</code> ke bot, bot akan otomatis meminta nomor telepon dengan tombol native <strong>KeyboardButtonRequestPhone</strong> dan mengirimkannya ke webhook.
                </Typography.Paragraph>

                <Form layout="vertical" onFinish={handleSaveStartConfig}>
                  <Form.Item label="Aktifkan Request Phone saat /start">
                    <Flex align="center" gap={12}>
                      <Switch
                        checked={startConfig.botRequestPhoneOnStart}
                        onChange={(checked) => setStartConfig('botRequestPhoneOnStart', checked)}
                      />
                      <Typography.Text type={startConfig.botRequestPhoneOnStart ? undefined : 'secondary'}>
                        {startConfig.botRequestPhoneOnStart
                          ? 'Aktif (Kirim tombol minta kontak saat /start)'
                          : 'Nonaktif'}
                      </Typography.Text>
                    </Flex>
                  </Form.Item>

                  {startConfig.botRequestPhoneOnStart && (
                    <>
                      <Form.Item
                        label="Pesan Selamat Datang & Permintaan Kontak"
                        help="Pesan teks yang dikirim bersama tombol permintaan nomor telepon"
                      >
                        <Input.TextArea
                          rows={2}
                          value={startConfig.botStartMessage}
                          onChange={(e) => setStartConfig('botStartMessage', e.target.value)}
                          placeholder="👋 Halo! Selamat datang. Silakan tekan tombol di bawah ini untuk membagikan nomor telepon Anda:"
                        />
                      </Form.Item>

                      <Form.Item
                        label="Teks Tombol Permintaan Kontak"
                        help="Teks yang muncul pada tombol keyboard di aplikasi Telegram"
                      >
                        <Input
                          prefix={<PhoneOutlined style={{ color: '#bfbfbf' }} />}
                          value={startConfig.botPhoneButtonText}
                          onChange={(e) => setStartConfig('botPhoneButtonText', e.target.value)}
                          placeholder="📱 Bagikan Nomor Telepon"
                        />
                      </Form.Item>

                      <Form.Item
                        label="Pesan Konfirmasi Setelah Kontak Diterima"
                        help="Pesan balasan setelah pengguna berhasil membagikan nomor teleponnya"
                      >
                        <Input.TextArea
                          rows={2}
                          value={startConfig.botPhoneReceivedMessage}
                          onChange={(e) => setStartConfig('botPhoneReceivedMessage', e.target.value)}
                          placeholder="✅ Terima kasih! Nomor telepon Anda telah berhasil diterima."
                        />
                      </Form.Item>
                    </>
                  )}

                  <Button
                    type="primary"
                    htmlType="submit"
                    loading={savingStartConfig}
                    icon={<SaveOutlined />}
                    style={{ width: '100%', marginTop: 8 }}
                  >
                    Simpan Pengaturan /start
                  </Button>
                </Form>
              </Card>
            )}

            {/* Menu Bot (Bot Commands) Card */}
            {device.hasBot && (
              <Card
                title={
                  <Space>
                    <AppstoreAddOutlined style={{ color: '#1890ff' }} />
                    <span>Menu Perintah Bot (Bot Commands)</span>
                  </Space>
                }
                extra={
                  menuConfig.botMenuSource === 'manual' ? (
                    <Space>
                      <Button
                        size="small"
                        icon={<ReloadOutlined />}
                        loading={loadingCommands}
                        onClick={fetchCommandsFromTelegram}
                      >
                        Tarik dari Telegram
                      </Button>
                      <Button
                        type="primary"
                        size="small"
                        icon={<PlusOutlined />}
                        onClick={handleOpenAddModal}
                      >
                        Tambah Menu
                      </Button>
                    </Space>
                  ) : (
                    <Tag color="cyan" icon={<GlobalOutlined />}>
                      Mode Dynamic URL (1m Cache)
                    </Tag>
                  )
                }
                bordered={false}
                style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}
              >
                {/* Source Selection Tabs / Radio */}
                <div style={{ marginBottom: 16, background: '#f8fafc', padding: '12px 16px', borderRadius: 8, border: '1px solid #e2e8f0' }}>
                  <Flex justify="space-between" align="center" wrap="wrap" gap={10}>
                    <div>
                      <Typography.Text strong style={{ fontSize: 13, display: 'block' }}>
                        Sumber Menu Perintah:
                      </Typography.Text>
                      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                        Pilih cara bot menentukan menu tombol <strong>[≡ Menu]</strong> di Telegram
                      </Typography.Text>
                    </div>
                    <Radio.Group
                      value={menuConfig.botMenuSource}
                      onChange={(e) => {
                        setMenuConfig('botMenuSource', e.target.value);
                      }}
                      optionType="button"
                      buttonStyle="solid"
                      size="middle"
                      options={[
                        { label: <><EditOutlined /> Buat Manual</>, value: 'manual' },
                        { label: <><GlobalOutlined /> Ambil dari URL / API</>, value: 'url' },
                      ]}
                    />
                  </Flex>
                </div>

                {menuConfig.botMenuSource === 'url' ? (
                  /* Form URL / API */
                  <Form layout="vertical" onFinish={handleSaveMenuConfig} style={{ marginTop: 8 }}>
                    <Alert
                      type="info"
                      showIcon
                      message="Menu Dinamis Berdasarkan Nomor HP Pengguna"
                      description={
                        <div style={{ fontSize: 12, lineHeight: 1.6, marginTop: 4 }}>
                          Saat pengguna berinteraksi dengan bot, bot akan memanggil endpoint URL di bawah ini untuk mengambil menu khusus pengguna tersebut (misal: menu pegawai, menu tim sales).
                          <ul style={{ paddingLeft: 18, margin: '6px 0' }}>
                            <li>Parameter yang dikirim: <code>?phone=+628123456789&user_id=123456789&device_id={device.id}</code></li>
                            <li>Hasil menu <strong>di-cache otomatis selama 1 menit (60 detik)</strong> per user agar cepat dan efisien.</li>
                            <li>Jika API gagal atau belum ada respon, menu default hanya <code>/start</code>.</li>
                          </ul>
                        </div>
                      }
                      style={{ marginBottom: 16 }}
                    />

                    <Form.Item
                      label="URL Endpoint Menu Bot (API URL)"
                      required
                      help="Contoh: https://kasir.doran.id/api/telegram-bot/menu"
                    >
                      <Input
                        prefix={<LinkOutlined style={{ color: '#bfbfbf' }} />}
                        placeholder="https://kasir.doran.id/api/telegram-bot/menu"
                        value={menuConfig.botMenuUrl}
                        onChange={(e) => setMenuConfig('botMenuUrl', e.target.value)}
                        size="large"
                      />
                    </Form.Item>

                    <div style={{ background: '#1f1f1f', borderRadius: 8, padding: 12, marginBottom: 16 }}>
                      <Typography.Text style={{ color: '#888', fontSize: 11, display: 'block', marginBottom: 6 }}>
                        Format Response JSON yang Diharapkan dari URL Anda:
                      </Typography.Text>
                      <pre style={{ color: '#a6e22e', fontSize: 11, margin: 0, overflowX: 'auto' }}>
{`[
  { "command": "start", "description": "Mulai bot & menu utama" },
  { "command": "omzet", "description": "Laporan Omzet Tim Sales" },
  { "command": "info", "description": "Informasi Data Pegawai" }
]`}
                      </pre>
                    </div>

                    <Button
                      type="primary"
                      htmlType="submit"
                      loading={savingMenuConfig}
                      icon={<SaveOutlined />}
                      size="large"
                      style={{ width: '100%' }}
                    >
                      Simpan Pengaturan Menu URL
                    </Button>
                  </Form>
                ) : (
                  /* Manual Commands Table */
                  <>
                    <Typography.Paragraph type="secondary" style={{ fontSize: 12, marginBottom: 12 }}>
                      Daftar menu perintah ini berlaku statis untuk semua pengguna dan akan muncul di tombol <strong>[≡ Menu]</strong> aplikasi Telegram.
                    </Typography.Paragraph>

                    <Table
                      rowKey="command"
                      dataSource={commands}
                      columns={commandColumns}
                      pagination={false}
                      size="small"
                      locale={{ emptyText: <Empty description="Belum ada menu perintah" /> }}
                      style={{ marginBottom: 16 }}
                    />

                    {/* Quick Templates */}
                    <div style={{ background: '#f8fafc', borderRadius: 8, padding: '10px 12px', marginBottom: 16 }}>
                      <Typography.Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 6 }}>
                        Template Cepat:
                      </Typography.Text>
                      <Flex wrap="wrap" gap={6}>
                        <Button size="small" onClick={() => handleAddTemplate('start', 'Mulai bot dan buka menu utama')}>
                          + /start
                        </Button>
                        <Button size="small" onClick={() => handleAddTemplate('saldo', 'Cek saldo dan riwayat transaksi')}>
                          + /saldo
                        </Button>
                        <Button size="small" onClick={() => handleAddTemplate('bantuan', 'Hubungi customer service')}>
                          + /bantuan
                        </Button>
                        <Button size="small" onClick={() => handleAddTemplate('info', 'Informasi akun & layanan')}>
                          + /info
                        </Button>
                        <Button size="small" onClick={() => handleAddTemplate('website', 'Kunjungi portal website')}>
                          + /website
                        </Button>
                      </Flex>
                    </div>

                    <Space direction="vertical" style={{ width: '100%' }} size="small">
                      <Button
                        type="primary"
                        icon={<SaveOutlined />}
                        size="large"
                        loading={savingCommands}
                        onClick={async () => {
                          await handleApplyCommandsToTelegram();
                          if (device.botMenuSource !== 'manual') {
                            setMenuConfig('botMenuSource', 'manual');
                            handleSaveMenuConfig();
                          }
                        }}
                        style={{ width: '100%' }}
                      >
                        Simpan & Terapkan ke Telegram
                      </Button>
                    </Space>
                  </>
                )}
              </Card>
            )}

            {/* Userbot Profile Card */}
            <Card
              title={
                <Space>
                  <UserOutlined style={{ color: '#1890ff' }} />
                  <span>Akun Userbot (User Account)</span>
                </Space>
              }
              bordered={false}
              style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}
            >
              <Flex align="center" gap={12}>
                <Badge status={device.hasSession ? 'success' : 'default'}>
                  <Avatar size={44} style={{ background: device.avatarColor || '#1890ff' }}>
                    {device.telegramFirstName?.charAt(0)?.toUpperCase() || device.name?.charAt(0)?.toUpperCase() || 'U'}
                  </Avatar>
                </Badge>
                <Flex vertical gap={0}>
                  <Typography.Text strong style={{ fontSize: 14 }}>
                    {device.telegramFirstName ? `${device.telegramFirstName} ${device.telegramLastName || ''}` : device.name}
                  </Typography.Text>
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    Nomor: {device.telegramPhone || device.phone}
                  </Typography.Text>
                </Flex>
                <Flex vertical align="flex-end" style={{ marginLeft: 'auto' }}>
                  <Typography.Text type="secondary" style={{ fontSize: 11 }}>
                    Status Session
                  </Typography.Text>
                  <Tag color={device.hasSession ? 'green' : 'default'} style={{ margin: 0, marginTop: 2 }}>
                    {device.hasSession ? 'Session Aktif' : 'Belum Login'}
                  </Tag>
                </Flex>
              </Flex>
            </Card>
          </Space>

          {/* Right Column: Documentation & Architecture Info */}
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            <Card
              title={
                <Space>
                  <ThunderboltOutlined style={{ color: '#fa8c16' }} />
                  <span>Panduan & Arsitektur Bot MTProto</span>
                </Space>
              }
              bordered={false}
              style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}
            >
              <Tabs
                defaultActiveKey="request_phone_guide"
                size="small"
                items={[
                  {
                    key: 'request_phone_guide',
                    label: 'Alur Request Phone',
                    children: (
                      <div style={{ fontSize: 13, lineHeight: '1.7', color: '#434343' }}>
                        <Typography.Paragraph>
                          Fitur <strong>Request Phone Number saat /start</strong> bekerja sebagai berikut:
                        </Typography.Paragraph>
                        <ol style={{ paddingLeft: 18, margin: '8px 0 16px' }}>
                          <li>Pengguna Telegram membuka bot dan menekan <strong>Start</strong> (atau mengirim <code>/start</code>).</li>
                          <li>Bot otomatis membalas dengan tombol <code>KeyboardButtonRequestPhone</code> (contoh: <em>[📱 Bagikan Nomor Telepon]</em>).</li>
                          <li>Saat pengguna menekan tombol, Telegram memunculkan konfirmasi bagikan nomor HP.</li>
                          <li>Nomor HP langsung diterima sistem, diteruskan ke <strong>Webhook Laravel</strong> dengan <code>sender.phone</code> terisi lengkap, dan bot mengirim pesan konfirmasi terima kasih.</li>
                        </ol>
                      </div>
                    ),
                  },
                  {
                    key: 'contact_payload',
                    label: 'Payload Webhook Kontak',
                    children: (
                      <div>
                        <Flex justify="space-between" align="center" style={{ marginBottom: 8 }}>
                          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                            Payload Kontak Masuk (application/json)
                          </Typography.Text>
                          <Button
                            size="small"
                            icon={<CopyOutlined />}
                            onClick={() => copyToClipboard(JSON.stringify(sampleContactPayload, null, 2), 'Payload Kontak')}
                          >
                            Salin JSON
                          </Button>
                        </Flex>
                        <pre
                          style={{
                            background: '#1f1f1f',
                            color: '#a6e22e',
                            padding: 14,
                            borderRadius: 8,
                            fontSize: 12,
                            maxHeight: 280,
                            overflowY: 'auto',
                            margin: 0,
                          }}
                        >
                          {JSON.stringify(sampleContactPayload, null, 2)}
                        </pre>
                      </div>
                    ),
                  },
                  {
                    key: 'menu_guide',
                    label: 'Cara Kerja Menu Bot',
                    children: (
                      <div style={{ fontSize: 13, lineHeight: '1.7', color: '#434343' }}>
                        <Typography.Paragraph>
                          Saat Anda menambahkan menu perintah di sini dan mengklik <strong>Simpan & Terapkan ke Telegram</strong>:
                        </Typography.Paragraph>
                        <ol style={{ paddingLeft: 18, margin: '8px 0 16px' }}>
                          <li>Sistem MTProto akan memanggil <code>BotsSetBotCommands</code> langsung ke Telegram server.</li>
                          <li>Pengguna yang membuka chat dengan bot <code>@{device.botUsername || 'bot'}</code> akan melihat tombol <strong>[≡ Menu]</strong> di pojok kiri bawah input teks Telegram.</li>
                          <li>Saat pengguna memilih salah satu menu (misal: <code>/start</code> atau <code>/saldo</code>), pesan perintah tersebut dikirim ke chat dan <strong>langsung diteruskan ke Webhook Laravel Anda</strong>.</li>
                        </ol>
                      </div>
                    ),
                  },
                  {
                    key: 'features',
                    label: 'Keunggulan MTProto Bot',
                    children: (
                      <div style={{ fontSize: 13, lineHeight: '1.7', color: '#434343' }}>
                        <Typography.Paragraph>
                          Integrasi <strong>Telegram Bot MTProto (gotd)</strong> memungkinkan bot Telegram bekerja berdampingan dengan <strong>Userbot Account</strong> pada satu device:
                        </Typography.Paragraph>
                        <ul style={{ paddingLeft: 18, margin: '8px 0 16px' }}>
                          <li>
                            <strong>Native Inline Buttons:</strong> Bot MTProto dapat membuat keyboard inline dengan <code>callback_data</code> asli Telegram yang muncul tepat di bawah pesan.
                          </li>
                          <li>
                            <strong>Realtime Callback Webhook:</strong> Saat tombol ditekan oleh pengguna di Telegram, event <code>callback_query</code> langsung diteruskan ke Webhook Laravel Anda.
                          </li>
                          <li>
                            <strong>Satu Dashboard:</strong> Kelola pesan personal userbot dan interaksi keyboard bot dari satu API dan backend terpadu.
                          </li>
                        </ul>
                      </div>
                    ),
                  },
                  {
                    key: 'botfather',
                    label: 'Cara Buat Bot Token',
                    children: (
                      <div style={{ fontSize: 13, lineHeight: '1.8', color: '#434343' }}>
                        <ol style={{ paddingLeft: 18, margin: 0 }}>
                          <li>Buka aplikasi Telegram dan cari akun resmi <code>@BotFather</code>.</li>
                          <li>Kirimkan perintah <code>/newbot</code>.</li>
                          <li>Beri nama bot Anda (misal: <em>Customer Service Bot</em>).</li>
                          <li>Tentukan username bot berakhiran <code>bot</code> (misal: <em>cs_layanan_bot</em>).</li>
                          <li>Salin <strong>HTTP API Token</strong> yang diberikan oleh BotFather dan tempelkan di form sebelah kiri.</li>
                        </ol>
                      </div>
                    ),
                  },
                  {
                    key: 'callback_payload',
                    label: 'Format Callback Webhook',
                    children: (
                      <div>
                        <Flex justify="space-between" align="center" style={{ marginBottom: 8 }}>
                          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                            Payload Callback Query (application/json)
                          </Typography.Text>
                          <Button
                            size="small"
                            icon={<CopyOutlined />}
                            onClick={() => copyToClipboard(JSON.stringify(sampleCallbackPayload, null, 2), 'Payload Callback')}
                          >
                            Salin JSON
                          </Button>
                        </Flex>
                        <pre
                          style={{
                            background: '#1f1f1f',
                            color: '#a6e22e',
                            padding: 14,
                            borderRadius: 8,
                            fontSize: 12,
                            maxHeight: 280,
                            overflowY: 'auto',
                            margin: 0,
                          }}
                        >
                          {JSON.stringify(sampleCallbackPayload, null, 2)}
                        </pre>
                      </div>
                    ),
                  },
                ]}
              />

              <Divider style={{ margin: '16px 0' }} />

              <Flex justify="space-between" align="center">
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  Ingin menguji pengiriman pesan dengan tombol?
                </Typography.Text>
                <Link href={`/devices/${device.id}/test-api`}>
                  <Button type="link" size="small" icon={<ApiOutlined />} style={{ padding: 0 }}>
                    Buka Test API →
                  </Button>
                </Link>
              </Flex>
            </Card>
          </Space>
        </div>
      </Space>

      {/* Modal Tambah / Edit Command */}
      <Modal
        title={editingIndex !== null ? 'Edit Menu Perintah Bot' : 'Tambah Menu Perintah Bot'}
        open={isModalOpen}
        onOk={handleSaveModal}
        onCancel={() => setIsModalOpen(false)}
        okText="Simpan Menu"
        cancelText="Batal"
      >
        <Form form={cmdForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="command"
            label="Perintah (Command)"
            rules={[
              { required: true, message: 'Nama perintah wajib diisi' },
              {
                pattern: /^[a-zA-Z0-9_]+$/,
                message: 'Hanya boleh berisi huruf, angka, dan underscore tanpa spasi atau slash',
              },
            ]}
            extra="Contoh: start, saldo, bantuan, riwayat (tanpa tanda slash /)"
          >
            <Input prefix="/" placeholder="start" />
          </Form.Item>

          <Form.Item
            name="description"
            label="Deskripsi Menu"
            rules={[{ required: true, message: 'Deskripsi menu wajib diisi' }]}
            extra="Deskripsi singkat yang akan tampil di menu Telegram (1-256 karakter)"
          >
            <Input placeholder="Mulai bot dan tampilkan menu utama" />
          </Form.Item>
        </Form>
      </Modal>
    </AppLayout>
  );
}
