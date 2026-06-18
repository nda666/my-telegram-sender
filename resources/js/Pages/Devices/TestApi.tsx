import {
    useRef,
    useState,
} from 'react';

import {
    Alert,
    Avatar,
    Badge,
    Button,
    Flex,
    Input,
    Radio,
    Select,
    Space,
    Tabs,
    Typography,
} from 'antd';

import {
    ApiOutlined,
    ArrowLeftOutlined,
    CopyOutlined,
    FileOutlined,
    IdcardOutlined,
    LinkOutlined,
    MobileOutlined,
    SendOutlined,
    UploadOutlined,
} from '@ant-design/icons';
import {
    Link,
    usePage,
} from '@inertiajs/react';

import AppLayout from '../../Layouts/AppLayout';

type Device = {
    id: number;
    name: string;
    phone: string;
    hasSession: boolean;
    telegramFirstName: string;
    telegramLastName: string;
    avatarColor: string;
    apiKey: string;
};

type PageProps = {
    device: Device;
    error?: string;
};

type Mode = 'text' | 'file' | 'url' | 'base64';
type TargetMode = 'chat_id' | 'phone';

type Result = { ok: boolean; status: number; data: unknown } | null;

const PEER_TYPES = ['user', 'chat', 'channel'];

export default function DeviceLiveTest() {
    const { device, error: propError } = usePage<PageProps>().props;

    const [mode, setMode] = useState<Mode>('text');
    const [targetMode, setTargetMode] = useState<TargetMode>('chat_id');

    const [chatId, setChatId] = useState('');
    const [phone, setPhone] = useState('');
    const [peerType, setPeerType] = useState('user');
    const [accessHash, setAccessHash] = useState('0');

    const [message, setMessage] = useState('');
    const [caption, setCaption] = useState('');
    const [mediaUrl, setMediaUrl] = useState('');
    const [file, setFile] = useState<File | null>(null);

    const [sending, setSending] = useState(false);
    const [result, setResult] = useState<Result>(null);
    const [error, setError] = useState<string | null>(propError || null);

    const fileInputRef = useRef<HTMLInputElement>(null);
    const apiKey = device.apiKey ?? '';

    const toBase64 = (f: File): Promise<string> =>
        new Promise((resolve, reject) => {
            const r = new FileReader();
            r.onload = () => resolve((r.result as string).split(',')[1]);
            r.onerror = reject;
            r.readAsDataURL(f);
        });

    const handleSend = async () => {
        if (targetMode === 'chat_id' && !chatId) { setError('chat_id wajib diisi'); return; }
        if (targetMode === 'phone' && !phone) { setError('Nomor HP wajib diisi'); return; }
        setSending(true);
        setError(null);
        setResult(null);

        try {
            let res: Response;

            if (mode === 'file' && file) {
                const fd = new FormData();
                if (targetMode === 'phone') {
                    fd.append('phone', phone);
                } else {
                    fd.append('chat_id', chatId);
                    fd.append('peer_type', peerType);
                    fd.append('access_hash', accessHash);
                }
                fd.append('caption', caption);
                fd.append('message', message);
                fd.append('file', file);
                res = await fetch('/api/send', {
                    method: 'POST',
                    headers: { 'X-Api-Key': apiKey },
                    body: fd,
                });
            } else {
                const body: Record<string, unknown> = {};
                if (targetMode === 'phone') {
                    body.phone = phone;
                } else {
                    body.chat_id = chatId;
                    body.peer_type = peerType;
                    body.access_hash = parseInt(accessHash) || 0;
                }
                if (mode === 'text') body.message = message;
                if (mode === 'url') { body.media_url = mediaUrl; body.caption = caption; }
                if (mode === 'base64' && file) {
                    body.media_base64 = await toBase64(file);
                    body.media_filename = file.name;
                    body.caption = caption;
                }
                res = await fetch('/api/send', {
                    method: 'POST',
                    headers: { 'X-Api-Key': apiKey, 'Content-Type': 'application/json' },
                    body: JSON.stringify(body),
                });
            }

            const data = await res.json().catch(() => ({}));
            setResult({ ok: res.ok, status: res.status, data });
        } catch (e: unknown) {
            setError(e instanceof Error ? e.message : 'Request gagal');
        } finally {
            setSending(false);
        }
    };

    const curlPreview = (() => {
        const h = `-H "X-Api-Key: ${apiKey}"`;
        const base = `curl -X POST /api/send \\\n  ${h}`;
        const target = targetMode === 'phone'
            ? `"phone":"${phone}"`
            : `"chat_id":"${chatId}","peer_type":"${peerType}","access_hash":${accessHash}`;
        const targetF = targetMode === 'phone'
            ? `-F "phone=${phone}"`
            : `-F "chat_id=${chatId}" -F "peer_type=${peerType}" -F "access_hash=${accessHash}"`;

        if (mode === 'text')
            return `${base} \\\n  -H "Content-Type: application/json" \\\n  -d '{${target},"message":"${message}"}'`;
        if (mode === 'url')
            return `${base} \\\n  -H "Content-Type: application/json" \\\n  -d '{${target},"media_url":"${mediaUrl}","caption":"${caption}"}'`;
        if (mode === 'file')
            return `${base} \\\n  ${targetF} \\\n  -F "caption=${caption}" -F "file=@${file?.name ?? 'file.jpg'}"`;
        return `${base} \\\n  -H "Content-Type: application/json" \\\n  -d '{${target},"media_base64":"<base64>","media_filename":"${file?.name ?? 'file.jpg'}","caption":"${caption}"}'`;
    })();

    const label = (text: string) => (
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>{text}</Typography.Text>
    );

    const canSend = targetMode === 'phone' ? !!phone : !!chatId;

    return (
        <AppLayout>
            <Space direction="vertical" size="large" style={{ width: '100%' }}>
                <Flex justify="space-between" align="center">
                    <Space>
                        <Link href={`/devices/${device.id}`}>
                            <Button type="link" icon={<ArrowLeftOutlined />} style={{ padding: 0 }}>
                                Kembali
                            </Button>
                        </Link>
                        <Typography.Title level={4} style={{ margin: 0 }}>
                            Test API — {device.name}
                        </Typography.Title>
                    </Space>
                </Flex>

                {error && <Alert message={error} type="error" showIcon closable onClose={() => setError(null)} />}

                <div style={{ display: 'flex', gap: 24, alignItems: 'flex-start' }}>
                    {/* Left: form */}
                    <div style={{ flex: 1, background: '#fff', borderRadius: 8, padding: 24, boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
                        {/* Device info */}
                        <Flex align="center" gap={12} style={{ marginBottom: 24, paddingBottom: 16, borderBottom: '1px solid #f0f0f0' }}>
                            <Badge status="success">
                                <Avatar size={40} style={{ background: device.avatarColor }}>
                                    {device.telegramFirstName?.charAt(0)?.toUpperCase()}
                                    {device.telegramLastName?.charAt(0)?.toUpperCase()}
                                </Avatar>
                            </Badge>
                            <Flex vertical gap={0}>
                                <Typography.Text strong>{device.telegramFirstName} {device.telegramLastName}</Typography.Text>
                                <Typography.Text type="secondary" style={{ fontSize: 12 }}>{device.phone}</Typography.Text>
                            </Flex>
                            <Flex vertical gap={0} style={{ marginLeft: 'auto', textAlign: 'right' }}>
                                {label('API Key')}
                                <Typography.Text code copyable style={{ fontSize: 11 }}>{apiKey}</Typography.Text>
                            </Flex>
                        </Flex>

                        {/* Target mode toggle */}
                        <Flex gap={8} align="center" style={{ marginBottom: 16 }}>
                            {label('Kirim ke')}
                            <Radio.Group
                                value={targetMode}
                                onChange={e => { setTargetMode(e.target.value); setChatId(''); setPhone(''); }}
                                optionType="button"
                                buttonStyle="solid"
                                size="small"
                                options={[
                                    { label: <><IdcardOutlined /> Chat ID</>, value: 'chat_id' },
                                    { label: <><MobileOutlined /> Nomor HP</>, value: 'phone' },
                                ]}
                            />
                        </Flex>

                        {/* Target fields */}
                        {targetMode === 'chat_id' ? (
                            <Flex gap={12} style={{ marginBottom: 16 }}>
                                <Flex vertical gap={4} style={{ flex: 1 }}>
                                    {label('Chat ID')}
                                    <Input value={chatId} onChange={e => setChatId(e.target.value)} placeholder="123456789" />
                                </Flex>
                                <Flex vertical gap={4} style={{ width: 130 }}>
                                    {label('Peer Type')}
                                    <Select value={peerType} onChange={setPeerType} options={PEER_TYPES.map(t => ({ value: t, label: t }))} />
                                </Flex>
                                <Flex vertical gap={4} style={{ width: 160 }}>
                                    {label('Access Hash (0 = group)')}
                                    <Input value={accessHash} onChange={e => setAccessHash(e.target.value)} />
                                </Flex>
                            </Flex>
                        ) : (
                            <Flex gap={12} style={{ marginBottom: 16 }}>
                                <Flex vertical gap={4} style={{ flex: 1 }}>
                                    {label('Nomor HP (format internasional)')}
                                    <Input
                                        value={phone}
                                        onChange={e => setPhone(e.target.value)}
                                        placeholder="+628123456789"
                                        prefix={<MobileOutlined />}
                                    />
                                </Flex>
                            </Flex>
                        )}

                        {/* Mode tabs */}
                        <Tabs
                            activeKey={mode}
                            onChange={k => { setMode(k as Mode); setFile(null); }}
                            size="small"
                            style={{ marginBottom: 8 }}
                            items={[
                                { key: 'text', label: <><SendOutlined /> Text</> },
                                { key: 'file', label: <><UploadOutlined /> File Upload</> },
                                { key: 'url', label: <><LinkOutlined /> Media URL</> },
                                { key: 'base64', label: <><FileOutlined /> Base64</> },
                            ]}
                        />

                        {/* Mode content */}
                        <Flex vertical gap={12}>
                            {mode === 'text' && (
                                <>
                                    {label('Message')}
                                    <Input.TextArea
                                        rows={3}
                                        value={message}
                                        onChange={e => setMessage(e.target.value)}
                                        placeholder="Hello bro!"
                                        onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend(); } }}
                                    />
                                </>
                            )}

                            {(mode === 'file' || mode === 'base64') && (
                                <>
                                    {label('File')}
                                    <input
                                        ref={fileInputRef}
                                        type="file"
                                        onChange={e => setFile(e.target.files?.[0] ?? null)}
                                        style={{ display: 'none' }}
                                    />
                                    <Flex gap={8} align="center">
                                        <Button icon={<UploadOutlined />} onClick={() => fileInputRef.current?.click()}>
                                            Pilih File
                                        </Button>
                                        {file && (
                                            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                                                {file.name} ({(file.size / 1024).toFixed(1)} KB)
                                            </Typography.Text>
                                        )}
                                    </Flex>
                                    {label('Caption')}
                                    <Input value={caption} onChange={e => setCaption(e.target.value)} placeholder="optional..." />
                                </>
                            )}

                            {mode === 'url' && (
                                <>
                                    {label('Media URL')}
                                    <Input
                                        value={mediaUrl}
                                        onChange={e => setMediaUrl(e.target.value)}
                                        placeholder="https://example.com/photo.jpg"
                                        prefix={<LinkOutlined />}
                                    />
                                    {label('Caption')}
                                    <Input value={caption} onChange={e => setCaption(e.target.value)} placeholder="optional..." />
                                </>
                            )}

                            <Button
                                type="primary"
                                icon={<SendOutlined />}
                                size="large"
                                loading={sending}
                                onClick={handleSend}
                                disabled={!canSend}
                            >
                                Kirim
                            </Button>
                        </Flex>
                    </div>

                    {/* Right: result + curl */}
                    <div style={{ width: 420, display: 'flex', flexDirection: 'column', gap: 16 }}>
                        {/* Result */}
                        <div style={{ background: '#fff', borderRadius: 8, padding: 20, boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
                            <Flex align="center" gap={8} style={{ marginBottom: 12 }}>
                                <ApiOutlined style={{ color: '#1890ff' }} />
                                <Typography.Text strong>Response</Typography.Text>
                                {result && (
                                    <Badge
                                        color={result.ok ? 'green' : 'red'}
                                        text={<span style={{ fontSize: 12 }}>{result.ok ? 'OK' : 'Error'} {result.status}</span>}
                                    />
                                )}
                            </Flex>
                            {result ? (
                                <pre style={{
                                    background: result.ok ? '#f6ffed' : '#fff2f0',
                                    border: `1px solid ${result.ok ? '#b7eb8f' : '#ffccc7'}`,
                                    borderRadius: 6,
                                    padding: 12,
                                    fontSize: 12,
                                    margin: 0,
                                    whiteSpace: 'pre-wrap',
                                    wordBreak: 'break-all',
                                    maxHeight: 200,
                                    overflowY: 'auto',
                                }}>
                                    {JSON.stringify(result.data, null, 2)}
                                </pre>
                            ) : (
                                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                                    Belum ada response. Kirim request dulu.
                                </Typography.Text>
                            )}
                        </div>

                        {/* cURL */}
                        <div style={{ background: '#1f1f1f', borderRadius: 8, padding: 20, boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
                            <Flex align="center" justify="space-between" style={{ marginBottom: 10 }}>
                                <Typography.Text style={{ color: '#888', fontSize: 12 }}>cURL</Typography.Text>
                                <Button
                                    size="small"
                                    icon={<CopyOutlined />}
                                    onClick={() => navigator.clipboard.writeText(curlPreview)}
                                    style={{ fontSize: 11 }}
                                >
                                    Copy
                                </Button>
                            </Flex>
                            <pre style={{ color: '#a6e22e', fontSize: 11, margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
                                {curlPreview}
                            </pre>
                        </div>
                    </div>
                </div>
            </Space>
        </AppLayout>
    );
}