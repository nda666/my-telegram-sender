import {
    useCallback,
    useEffect,
    useState,
} from 'react';

import {
    App,
    Button,
    Col,
    Flex,
    Form,
    Input,
    Modal,
    Popconfirm,
    Row,
    Space,
    Table,
    Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';

import { router } from '@inertiajs/react';

import AppLayout from '../../Layouts/AppLayout';

const { Title } = Typography;

interface Contact {
    user_id: number;
    access_hash: number;
    first_name: string;
    last_name: string;
    username: string;
    phone: string;
}

interface Props {
    contacts: Contact[];
    deviceID: number;
    total: number;
    page: number;
    pageSize: number;
    searchName: string;
    searchUsername: string;
    searchPhone: string;
}

interface FormValues {
    phone: string;
    first_name: string;
    last_name: string;
}

export default function ContactsIndex({
    contacts,
    deviceID,
    total,
    page,
    pageSize,
    searchName: initName,
    searchUsername: initUsername,
    searchPhone: initPhone,
}: Props) {
    const { message } = App.useApp();
    const [antdForm] = Form.useForm<FormValues>();

    // Local state hanya untuk optimistic update (add/edit/delete)
    const [rows, setRows] = useState<Contact[]>(contacts);
    const [totalRows, setTotalRows] = useState(total);

    const [loading, setLoading] = useState(false);
    const [modal, setModal] = useState<{ open: boolean; editing: Contact | null }>({
        open: false, editing: null,
    });

    // Search inputs — dikontrol lokal, dikirim ke server saat submit / pagination
    const [sName, setSName] = useState(initName);
    const [sUsername, setSUsername] = useState(initUsername);
    const [sPhone, setSPhone] = useState(initPhone);

    // Kirim query ke server via Inertia (GET, preserve scroll)
    const navigate = useCallback((overrides: Record<string, unknown> = {}) => {
        router.get(
            `/devices/${deviceID}/contacts`,
            {
                device_id: deviceID,
                page: page,
                page_size: pageSize,
                search_name: sName,
                search_username: sUsername,
                search_phone: sPhone,
                ...overrides,
            },
            { preserveScroll: true, preserveState: true },
        );
    }, [deviceID, page, pageSize, sName, sUsername, sPhone]);

    useEffect(() => {
        setRows(contacts);
    }, [contacts]);
    const handleSearch = () => navigate({ page: 1 });

    const handlePageChange = (p: number, ps: number) => navigate({ page: p, page_size: ps });

    // ---- CRUD (sama seperti sebelumnya, tapi update state lokal) ----
    const openCreate = () => { antdForm.resetFields(); setModal({ open: true, editing: null }); };
    const openEdit = (c: Contact) => {
        antdForm.setFieldsValue({ phone: c.phone, first_name: c.first_name, last_name: c.last_name });
        setModal({ open: true, editing: c });
    };
    const handleCancel = () => { setModal({ open: false, editing: null }); antdForm.resetFields(); };

    const handleSubmit = async () => {
        try {
            const values = await antdForm.validateFields();
            setLoading(true);

            const isEdit = !!modal.editing;
            const url = isEdit ? `/devices/${deviceID}/contacts` : `/devices/${deviceID}/contacts`;
            const method = isEdit ? 'PUT' : 'POST';

            const res = await fetch(url, {
                method,
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ device_id: deviceID, ...values }),
            });
            const data = await res.json();

            if (!res.ok) { message.error(data.error ?? 'Terjadi kesalahan'); return; }

            const saved: Contact = data.data;
            if (isEdit) {
                setRows(cs => cs.map(c => c.user_id === saved.user_id ? saved : c));
                message.success('Kontak berhasil diperbarui');
            } else {
                setRows(cs => [...cs, saved]);
                setTotalRows(t => t + 1);
                message.success('Kontak berhasil ditambahkan');
            }
            setModal({ open: false, editing: null });
        } catch (err) {
            console.error(err);
        } finally {
            setLoading(false);
        }
    };

    const confirmDelete = async (target: Contact) => {
        const res = await fetch(`/devices/${deviceID}/contacts/${target.user_id}`, {
            method: 'DELETE',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ device_id: deviceID, access_hash: target.access_hash }),
        });
        if (res.ok) {
            setRows(cs => cs.filter(c => c.user_id !== target.user_id));
            setTotalRows(t => t - 1);
            message.success('Kontak berhasil dihapus');
        } else {
            const data = await res.json();
            message.error(data.error ?? 'Gagal menghapus kontak');
        }
    };

    const columns: ColumnsType<Contact> = [
        {
            title: 'Name', key: 'name',
            render: (_, r) => `${r.first_name} ${r.last_name || ''}`.trim(),
        },
        {
            title: 'Username', dataIndex: 'username', key: 'username',
            render: t => t ? `@${t}` : '-',
        },
        {
            title: 'Phone', dataIndex: 'phone', key: 'phone',
            render: t => t || '-',
        },
        {
            title: 'User ID', dataIndex: 'user_id', key: 'user_id',
            render: t => <span className="font-mono text-xs text-gray-400">{t}</span>,
        },
        {
            title: 'Actions', key: 'actions', align: 'right',
            render: (_, record) => (
                record.username ?
                    <Space size="middle">
                        <Button type="link" size="small" onClick={() => openEdit(record)}>Edit</Button>
                        <Popconfirm
                            title="Hapus Kontak"
                            description={`Hapus ${record.first_name} ${record.last_name || ''} dari kontak Telegram?`}
                            onConfirm={() => confirmDelete(record)}
                            okText="Yes" cancelText="No" okButtonProps={{ danger: true }}
                        >
                            <Button type="link" danger size="small">Delete</Button>
                        </Popconfirm>
                    </Space>
                    : <small><i>no user found in Telegram</i></small>
            ),
        },
    ];

    return (
        <AppLayout>
            <div className="max-w-4xl mx-auto mt-8 px-4">

                <Flex justify="space-between" align="center" style={{ marginBottom: 16 }}>
                    <Title level={2} style={{ margin: 0 }}>
                        Telegram Contacts
                    </Title>
                </Flex>
                {/* Search bar */}
                <Form className="flex gap-2 mb-4" onFinish={handleSearch} >
                    <Row >
                        <Col span={12}>
                            <Form.Item name="name" label="Name" layout="horizontal"
                                labelCol={{ span: 6 }}
                                wrapperCol={{ span: 18 }}>
                                <Input
                                    placeholder="Search name…"
                                    value={sName}
                                    onChange={e => setSName(e.target.value)}
                                    onPressEnter={handleSearch}
                                    allowClear
                                    style={{ flex: 2 }}
                                />
                            </Form.Item>
                            <Form.Item name="username" label="Username" layout="horizontal"
                                labelCol={{ span: 6 }}
                                wrapperCol={{ span: 18 }}>     <Input
                                    placeholder="Search username…"
                                    value={sUsername}
                                    onChange={e => setSUsername(e.target.value)}
                                    onPressEnter={handleSearch}
                                    allowClear
                                    style={{ flex: 2 }}
                                /></Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item name="phone" label="Phone" layout="horizontal"
                                labelCol={{ span: 6 }}
                                wrapperCol={{ span: 18 }}>
                                <Input
                                    placeholder="Search phone…"
                                    value={sPhone}
                                    onChange={e => setSPhone(e.target.value)}
                                    onPressEnter={handleSearch}
                                    allowClear
                                    style={{ flex: 2 }}
                                />
                            </Form.Item>
                            <Form.Item layout="horizontal" wrapperCol={{ span: 18, offset: 6 }}>

                                <Button block type="primary" onClick={handleSearch}>Search</Button>
                            </Form.Item>
                        </Col>
                    </Row>
                </Form>
                <Flex justify="space-between" align="center" style={{ marginBottom: 16 }}>
                    <Button type="primary" onClick={openCreate}>
                        + Add Contact
                    </Button>
                </Flex>
                <Table
                    dataSource={rows}
                    columns={columns}
                    rowKey="user_id"
                    scroll={{ y: 450 }}
                    pagination={{
                        current: page,
                        pageSize: pageSize,
                        total: totalRows,
                        showSizeChanger: true,
                        onChange: handlePageChange,
                    }}
                    locale={{ emptyText: 'No contacts found.' }}
                />
            </div>

            <Modal
                title={modal.editing ? 'Edit Contact' : 'Add Contact'}
                open={modal.open}
                onOk={handleSubmit}
                onCancel={handleCancel}
                confirmLoading={loading}
                okText="Save"
                destroyOnClose
            >
                <Form form={antdForm} layout="vertical" className="mt-4">
                    <Form.Item
                        label="Phone" name="phone"
                        rules={[{ required: true, message: 'Phone number wajib diisi!' }]}
                        help={modal.editing ? 'Phone number tidak bisa diubah' : undefined}
                    >
                        <Input placeholder="+628123456789" disabled={!!modal.editing} />
                    </Form.Item>
                    <Form.Item label="First Name" name="first_name" rules={[{ required: true, message: 'First name wajib diisi!' }]}>
                        <Input />
                    </Form.Item>
                    <Form.Item label="Last Name" name="last_name">
                        <Input />
                    </Form.Item>
                </Form>
            </Modal>
        </AppLayout>
    );
}