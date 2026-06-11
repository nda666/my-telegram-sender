import { useState } from 'react';

import {
    App,
    Button,
    Card,
    Flex,
    Form,
    Input,
    Typography,
} from 'antd';

import AppLayout from '../../Layouts/AppLayout';

const { Title, Text } = Typography;

export default function ChangePassword() {
    const { message } = App.useApp();
    const [form] = Form.useForm();
    const [loading, setLoading] = useState(false);

    const submit = async (values: any) => {
        setLoading(true);
        try {
            const res = await fetch("/settings/password", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    "X-Inertia": "true",
                    "X-XSRF-TOKEN": getCookie("XSRF-TOKEN") ?? "",
                },
                body: JSON.stringify(values),
            });

            const data = await res.json();

            if (!res.ok) {
                message.error(data.error ?? "Something went wrong");
            } else {
                message.success("Password updated successfully!");
                form.resetFields();
            }
        } catch (error) {
            message.error("Failed to connect to the server");
        } finally {
            setLoading(false);
        }
    };

    return (
        <AppLayout>
            {/* 1. Container utama berpusat di tengah halaman secara horizontal */}
            <Flex style={{ width: '100%' }} vertical>
                <div style={{ marginBottom: 24 }}>
                    <Title level={3} style={{ margin: 0, fontWeight: 600 }}>
                        Change Password
                    </Title>

                </div>
                {/* 2. Membatasi lebar maksimal form (max-width) agar tidak melar di desktop */}
                <Card style={{ width: '100%' }}>

                    {/* Header Section dengan deskripsi singkat */}


                    {/* 3. Card dengan styling bayangan halus bawaan token Antd 6 */}
                    <div
                        style={{
                            maxWidth: 540
                        }}
                    >
                        <div style={{ marginBottom: 24 }}>
                            <Text type="secondary">
                                Ensure your account is using a long, random password to stay secure.
                            </Text>
                        </div>
                        <Form
                            form={form}
                            onFinish={submit}
                            requiredMark={false} // Menghilangkan tanda bintang merah jika dirasa terlalu ramai
                        >
                            <Form.Item
                                label="Current Password"
                                name="current_password"

                                layout="horizontal"
                                labelCol={{ span: 8 }}
                                wrapperCol={{ span: 16 }}
                                rules={[{ required: true, message: 'Please input your current password!' }]}
                            >
                                <Input.Password size="large" placeholder="Enter current password" />
                            </Form.Item>

                            <Form.Item
                                label="New Password"
                                name="new_password"

                                layout="horizontal"
                                labelCol={{ span: 8 }}
                                wrapperCol={{ span: 16 }}
                                rules={[
                                    { required: true, message: 'Please input your new password!' },
                                    { min: 8, message: 'Password must be at least 8 characters!' }
                                ]}
                            >
                                <Input.Password size="large" placeholder="Enter new password" />
                            </Form.Item>

                            <Form.Item
                                label="Confirm New Password"
                                name="password_confirmation"
                                dependencies={['new_password']}

                                layout="horizontal"
                                labelCol={{ span: 8 }}
                                wrapperCol={{ span: 16 }}
                                rules={[
                                    { required: true, message: 'Please confirm your new password!' },
                                    ({ getFieldValue }) => ({
                                        validator(_, value) {
                                            if (!value || getFieldValue('new_password') === value) {
                                                return Promise.resolve();
                                            }
                                            return Promise.reject(new Error('The two passwords do not match!'));
                                        },
                                    }),
                                ]}
                            >
                                <Input.Password size="large" placeholder="Confirm new password" />
                            </Form.Item>

                            {/* Spacing penutup yang pas untuk tombol */}
                            <Form.Item style={{ marginBottom: 0, marginTop: 8 }}>
                                <Button
                                    type="primary"
                                    htmlType="submit"
                                    loading={loading}
                                    size="large"
                                    block
                                >
                                    Update Password
                                </Button>
                            </Form.Item>
                        </Form>
                    </div>
                </Card>
            </Flex>
        </AppLayout >
    );
}

function getCookie(name: string) {
    return document.cookie
        .split("; ")
        .find((r) => r.startsWith(name + "="))
        ?.split("=")[1];
}