import React, {
  useEffect,
  useRef,
  useState,
} from 'react';

import {
  Alert,
  Avatar,
  Badge,
  Button,
  Empty,
  Flex,
  Image,
  Input,
  List,
  Result,
  Space,
  Spin,
  Typography,
} from 'antd';

import {
  ArrowLeftOutlined,
  FileOutlined,
  MessageOutlined,
  NotificationOutlined,
  ReloadOutlined,
  SendOutlined,
  TeamOutlined,
  UserOutlined,
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
};

type ChatItem = {
  id: string;
  type: 'user' | 'chat' | 'channel';
  name: string;
  username?: string;
  phone?: string;
  lastMessage?: string;
  lastMessageTime?: string;
  unreadCount: number;
  accessHash?: string;
};

type Message = {
  id: number;
  senderId: string;
  senderName: string;
  text: string;
  out: boolean;
  time: string;
  mediaType?: 'photo' | 'video' | 'voice' | 'audio' | 'sticker' | 'gif' | 'document';
};

type PageProps = {
  device: Device;
  chats: ChatItem[];
  error?: string;
};

export default function DeviceInbox() {
  const { device, chats, error: propError } = usePage<PageProps>().props;
  const [selectedChat, setSelectedChat] = useState<ChatItem | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [sending, setSending] = useState(false);
  const [inputText, setInputText] = useState('');
  const [error, setError] = useState<string | null>(propError || null);

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesBodyRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    if (selectedChat) {
      setMessages([]);
      setHasMore(true);
      fetchMessages();
    } else {
      setMessages([]);
    }
  }, [selectedChat]);

  useEffect(() => {
    scrollToBottom();
  }, [selectedChat]); // scroll ke bawah hanya saat ganti chat

  const buildURL = (chat: ChatItem, offsetID?: number) => {
    let url = `/devices/${device.id}/inbox/messages?type=${chat.type}&peer_id=${chat.id}&access_hash=${chat.accessHash || ''}`;
    if (offsetID) url += `&offset_id=${offsetID}`;
    return url;
  };

  const fetchMessages = async () => {
    if (!selectedChat) return;
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(buildURL(selectedChat));
      const data = await response.json();
      if (data.error) {
        setError(data.error);
      } else {
        const msgs: Message[] = data.messages || [];
        setMessages(msgs);
        setHasMore(msgs.length === 50);
        // scroll ke bawah setelah load awal
        setTimeout(() => messagesEndRef.current?.scrollIntoView({ behavior: 'auto' }), 50);
      }
    } catch {
      setError('Gagal memuat pesan. Pastikan koneksi internet aktif.');
    } finally {
      setLoading(false);
    }
  };

  const fetchMoreMessages = async () => {
    if (!selectedChat || loadingMore || !hasMore || messages.length === 0) return;
    setLoadingMore(true);
    const oldestID = messages[0].id;
    const body = messagesBodyRef.current;
    const prevScrollHeight = body?.scrollHeight ?? 0;
    try {
      const response = await fetch(buildURL(selectedChat, oldestID));
      const data = await response.json();
      if (!data.error) {
        const older: Message[] = data.messages || [];
        setHasMore(older.length === 50);
        setMessages((prev) => [...older, ...prev]);
        // pertahankan posisi scroll setelah prepend
        requestAnimationFrame(() => {
          if (body) {
            body.scrollTop = body.scrollHeight - prevScrollHeight;
          }
        });
      }
    } catch {
      // silent
    } finally {
      setLoadingMore(false);
    }
  };

  const handleScroll = () => {
    const body = messagesBodyRef.current;
    if (!body) return;
    if (body.scrollTop < 80) {
      fetchMoreMessages();
    }
  };

  const handleSendMessage = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedChat || !inputText.trim() || sending) return;

    setSending(true);
    const textToSend = inputText.trim();
    setInputText('');

    try {
      const response = await fetch(`/devices/${device.id}/inbox/send`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          type: selectedChat.type,
          peer_id: selectedChat.id,
          access_hash: selectedChat.accessHash,
          message: textToSend,
        }),
      });

      const data = await response.json();
      if (data.error) {
        setError(data.error);
      } else {
        const newMessage: Message = {
          id: Date.now(),
          senderId: '0',
          senderName: 'Anda',
          text: textToSend,
          out: true,
          time: new Date().toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' }),
        };
        setMessages((prev) => [...prev, newMessage]);
      }
    } catch {
      setError('Gagal mengirim pesan.');
    } finally {
      setSending(false);
    }
  };

  const getChatIcon = (type: ChatItem['type']) => {
    switch (type) {
      case 'chat':
        return <Avatar style={{ backgroundColor: '#1890ff' }} icon={<TeamOutlined />} />;
      case 'channel':
        return <Avatar style={{ backgroundColor: '#faad14' }} icon={<NotificationOutlined />} />;
      default:
        return <Avatar style={{ backgroundColor: '#52c41a' }} icon={<UserOutlined />} />;
    }
  };

  const mediaURL = (msg: Message, chat: ChatItem) =>
    `/devices/${device.id}/inbox/media?msg_id=${msg.id}&type=${chat.type}&peer_id=${chat.id}&access_hash=${chat.accessHash || ''}&media_type=${msg.mediaType || ''}`;

  const renderMessageContent = (msg: Message) => {
    const isOut = msg.out;
    const textColor = isOut ? '#fff' : '#000';
    const chat = selectedChat!;

    if (msg.mediaType === 'photo') {
      return (
        <Flex vertical gap="middle">
          <Image
            src={mediaURL(msg, chat)}
            alt="photo"
            width={200}
            preview={{
              mask: 'Preview',
            }}
          />
          {msg.text && (
            <Typography.Text style={{ color: textColor }}>{msg.text}</Typography.Text>
          )}
        </Flex>
      );
    }

    if (msg.mediaType === 'sticker') {
      return (
        <img
          src={mediaURL(msg, chat)}
          alt="sticker"
          style={{ maxWidth: 120, display: 'block' }}
        />
      );
    }

    if (msg.mediaType === 'gif') {
      return (
        <video
          src={mediaURL(msg, chat)}
          autoPlay
          loop
          muted
          playsInline
          style={{ maxWidth: '200px', borderRadius: 8, display: 'block' }}
        />
      );
    }

    if (msg.mediaType === 'video') {
      return (
        <div>
          <video
            src={mediaURL(msg, chat)}
            controls
            style={{ maxWidth: '100%', borderRadius: 8, display: 'block', marginBottom: msg.text ? 6 : 0 }}
          />
          {msg.text && (
            <Typography.Text style={{ color: textColor }}>{msg.text}</Typography.Text>
          )}
        </div>
      );
    }

    if (msg.mediaType === 'voice' || msg.mediaType === 'audio') {
      return (
        <div>
          <audio
            src={mediaURL(msg, chat)}
            controls
            style={{ width: '100%', marginBottom: msg.text ? 6 : 0 }}
          />
          {msg.text && (
            <Typography.Text style={{ color: textColor }}>{msg.text}</Typography.Text>
          )}
        </div>
      );
    }

    if (msg.mediaType === 'document') {
      return (
        <div>
          <a
            href={mediaURL(msg, chat)}
            target="_blank"
            rel="noreferrer"
            style={{ color: isOut ? '#fff' : '#1890ff', display: 'flex', alignItems: 'center', gap: 6 }}
          >
            <FileOutlined />
            <span>Unduh Dokumen</span>
          </a>
          {msg.text && (
            <Typography.Text style={{ color: textColor, display: 'block', marginTop: 4 }}>
              {msg.text}
            </Typography.Text>
          )}
        </div>
      );
    }

    // plain text (no media)
    return (
      <Typography.Text style={{ color: textColor }}>
        {msg.text || <span style={{ opacity: 0.5 }}>[pesan tidak didukung]</span>}
      </Typography.Text>
    );
  };

  // bubble background transparan untuk sticker/gif
  const bubbleStyle = (msg: Message): React.CSSProperties => {
    const isTransparent = msg.mediaType === 'sticker' || msg.mediaType === 'gif';
    return {
      background: isTransparent ? 'transparent' : msg.out ? '#1890ff' : '#fff',
      color: msg.out ? '#fff' : '#000',
      padding: isTransparent ? '0' : '10px 16px',
      borderRadius: msg.out ? '16px 16px 0 16px' : '16px 16px 16px 0',
      boxShadow: isTransparent ? 'none' : '0 1px 2px rgba(0,0,0,0.1)',
      wordBreak: 'break-word',
    };
  };

  return (
    <AppLayout>
      <Space orientation="vertical" size="large" style={{ width: '100%', height: 'calc(100vh - 120px)' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Link href="/devices">
              <Button type="link" icon={<ArrowLeftOutlined />} style={{ padding: 0 }}>
                Kembali
              </Button>
            </Link>
            <Typography.Title level={4} style={{ margin: 0 }}>
              Inbox — {device.name}
            </Typography.Title>
          </Space>
          {selectedChat && (
            <Button icon={<ReloadOutlined />} onClick={fetchMessages} loading={loading}>
              Segarkan
            </Button>
          )}
        </div>

        {error && !selectedChat && (
          <Alert description={error} type="error" showIcon style={{ marginBottom: 16 }} />
        )}

        <div
          style={{
            display: 'flex',
            height: 'calc(100vh - 160px)',
            background: '#f5f5f5',
            borderRadius: 8,
            overflow: 'hidden',
            boxShadow: '0 2px 8px rgba(0,0,0,0.06)',
          }}
        >
          {/* Left panel */}
          <div
            style={{
              width: '320px',
              borderRight: '1px solid #e8e8e8',
              background: '#fff',
              display: 'flex',
              flexDirection: 'column',
            }}
          >
            <Flex
              align="center"
              justify="space-between"
              style={{
                padding: '14px 14px',
                borderBottom: '1px solid #e8e8e8',
                background: '#fafafa',
              }}
            >
              <Space size={12}>
                <Badge status="success">
                  <Avatar size={48} style={{ background: device.avatarColor }}>
                    {device.telegramFirstName?.charAt(0)?.toUpperCase()}
                    {device.telegramLastName?.charAt(0)?.toUpperCase()}
                  </Avatar>
                </Badge>

                <Flex vertical gap={0}>
                  <Typography.Text strong style={{ fontSize: 15 }}>
                    {`${device.telegramFirstName} ${device.telegramLastName}`}
                  </Typography.Text>

                  <Typography.Text
                    type="secondary"
                    style={{ fontSize: 12 }}
                  >
                    {device.phone}
                  </Typography.Text>
                </Flex>
              </Space>

              <Flex vertical align="flex-end">
                <Typography.Text strong>
                  {chats?.length ?? 0}
                </Typography.Text>

                <Typography.Text
                  type="secondary"
                  style={{ fontSize: 12 }}
                >
                  Percakapan
                </Typography.Text>
              </Flex>
            </Flex>
            <div style={{ flex: 1, overflowY: 'auto' }}>
              {chats?.length === 0 ? (
                <div style={{ padding: 32, textAlign: 'center' }}>
                  <Empty description="Tidak ada percakapan" image={Empty.PRESENTED_IMAGE_SIMPLE} />
                </div>
              ) : (
                <List
                  itemLayout="horizontal"
                  dataSource={chats || []}
                  renderItem={(item) => (
                    <List.Item
                      onClick={() => setSelectedChat(item)}
                      style={{
                        padding: '12px 16px',
                        cursor: 'pointer',
                        background: selectedChat?.id === item.id ? '#e6f7ff' : 'transparent',
                        transition: 'background 0.3s',
                      }}
                      className="chat-list-item"
                    >
                      <List.Item.Meta
                        avatar={getChatIcon(item.type)}
                        title={
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <Typography.Text strong ellipsis style={{ maxWidth: '160px' }}>
                              {item.name}
                            </Typography.Text>
                            <Typography.Text type="secondary" style={{ fontSize: '11px' }}>
                              {item.lastMessageTime}
                            </Typography.Text>
                          </div>
                        }
                        description={
                          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                            <Typography.Text
                              type="secondary"
                              ellipsis
                              style={{ maxWidth: '180px', fontSize: '12px' }}
                            >
                              {item.lastMessage || (item.username ? `@${item.username}` : '')}
                            </Typography.Text>
                            {item.unreadCount > 0 && (
                              <Badge count={item.unreadCount} style={{ backgroundColor: '#ff4d4f' }} />
                            )}
                          </div>
                        }
                      />
                    </List.Item>
                  )}
                />
              )}
            </div>
          </div>

          {/* Right panel */}
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', background: '#fff' }}>
            {selectedChat ? (
              <>
                {/* Header */}
                <div
                  style={{
                    padding: '16px 24px',
                    borderBottom: '1px solid #e8e8e8',
                    background: '#fafafa',
                    display: 'flex',
                    alignItems: 'center',
                  }}
                >
                  {getChatIcon(selectedChat.type)}
                  <div style={{ marginLeft: 16 }}>
                    <Typography.Text strong style={{ fontSize: 16, display: 'block' }}>
                      {selectedChat.name}
                    </Typography.Text>
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      {selectedChat.type === 'user'
                        ? selectedChat.phone || (selectedChat.username ? `@${selectedChat.username}` : 'Kontak Telegram')
                        : `${selectedChat.type} Group`}
                    </Typography.Text>
                  </div>
                </div>

                {/* Messages */}
                <div
                  ref={messagesBodyRef}
                  onScroll={handleScroll}
                  style={{ flex: 1, padding: '24px', overflowY: 'auto', background: '#f0f2f5' }}
                >
                  {loading ? (
                    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
                      <Spin size="large" tip="Memuat riwayat pesan..." />
                    </div>
                  ) : (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                      {loadingMore && (
                        <div style={{ textAlign: 'center', padding: '8px 0' }}>
                          <Spin size="small" tip="Memuat pesan lama..." />
                        </div>
                      )}
                      {!hasMore && messages.length > 0 && (
                        <div style={{ textAlign: 'center', padding: '4px 0' }}>
                          <Typography.Text type="secondary" style={{ fontSize: 11 }}>Tidak ada pesan lebih lama</Typography.Text>
                        </div>
                      )}
                      {error && (
                        <Alert description={error} type="error" showIcon closable onClose={() => setError(null)} />
                      )}
                      {messages.length === 0 ? (
                        <div style={{ textAlign: 'center', padding: 48 }}>
                          <Typography.Text type="secondary">Belum ada pesan di percakapan ini.</Typography.Text>
                        </div>
                      ) : (
                        messages.map((msg) => (
                          <div
                            key={msg.id}
                            style={{ alignSelf: msg.out ? 'flex-end' : 'flex-start', maxWidth: '70%' }}
                          >
                            {!msg.out && selectedChat.type !== 'user' && (
                              <div style={{ fontSize: 11, color: '#8c8c8c', marginBottom: 2, marginLeft: 8 }}>
                                {msg.senderName}
                              </div>
                            )}
                            <div style={bubbleStyle(msg)}>
                              {renderMessageContent(msg)}
                              {msg.mediaType !== 'sticker' && (
                                <div
                                  style={{
                                    fontSize: 9,
                                    color: msg.out ? 'rgba(255,255,255,0.7)' : '#8c8c8c',
                                    textAlign: 'right',
                                    marginTop: 4,
                                  }}
                                >
                                  {msg.time}
                                </div>
                              )}
                            </div>
                          </div>
                        ))
                      )}
                      <div ref={messagesEndRef} />
                    </div>
                  )}
                </div>

                {/* Footer */}
                <div style={{ padding: '16px 24px', borderTop: '1px solid #e8e8e8', background: '#fff' }}>
                  <form onSubmit={handleSendMessage} style={{ display: 'flex', gap: 12 }}>
                    <Input
                      value={inputText}
                      onChange={(e) => setInputText(e.target.value)}
                      placeholder="Ketik pesan..."
                      size="large"
                      disabled={loading || sending}
                      autoFocus
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' && !e.shiftKey) {
                          e.preventDefault();
                          handleSendMessage(e);
                        }
                      }}
                    />
                    <Button
                      type="primary"
                      htmlType="submit"
                      icon={<SendOutlined />}
                      size="large"
                      loading={sending}
                      disabled={!inputText.trim()}
                    >
                      Kirim
                    </Button>
                  </form>
                </div>
              </>
            ) : (
              <div style={{ flex: 1, display: 'flex', justifyContent: 'center', alignItems: 'center', padding: 48 }}>
                <Result
                  icon={<MessageOutlined style={{ fontSize: 64, color: '#1890ff' }} />}
                  title="Inbox Telegram"
                  subTitle="Pilih percakapan di sebelah kiri untuk melihat riwayat pesan dan mulai berkirim pesan langsung."
                />
              </div>
            )}
          </div>
        </div>
      </Space>
    </AppLayout>
  );
}