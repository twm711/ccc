import { useState, useEffect } from 'react';
import { Card, Descriptions, Tag, Tabs, Spin, Empty, Space } from 'antd';
import { UserOutlined, PhoneOutlined, HistoryOutlined, ApiOutlined } from '@ant-design/icons';
import api from '../../api/client';

interface CustomerInfo {
  name: string;
  phone: string;
  company: string;
  level: string;
  last_contact: string;
  notes: string;
  tags: string[];
  history: { id: number; date: string; type: string; summary: string }[];
}

interface ScreenPopResponse {
  customer: CustomerInfo | null;
  urls?: string[];
  ivr_context?: Record<string, string>;
  iframe_url?: string;
  interactions?: { id: number; date: string; type: string; summary: string }[];
}

export default function ScreenPopPanel({ callerNumber, callId }: { callerNumber?: string; callId?: number }) {
  const [customer, setCustomer] = useState<CustomerInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [iframeUrls, setIframeUrls] = useState<string[]>([]);
  const [ivrContext, setIvrContext] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!callerNumber && !callId) return;
    setLoading(true);
    const params: Record<string, string | number> = {};
    if (callId) params.call_id = callId;
    else if (callerNumber) params.phone = callerNumber;
    api.get('/screen-pop/lookup', { params })
      .then((res) => {
        const data = res.data as ScreenPopResponse;
        setCustomer(data?.customer || null);
        setIvrContext(data?.ivr_context || {});
        const urls = data?.urls || [];
        const combined = urls.length > 0 ? urls : (data?.iframe_url ? [data.iframe_url] : []);
        setIframeUrls(combined);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [callerNumber, callId]);

  if (loading) return <Card size="small"><Spin style={{ display: 'block', padding: 40 }} /></Card>;
  if (!callerNumber && !callId) return null;

  return (
    <Card title={<Space><UserOutlined /> 来电弹屏</Space>} size="small">
      <Tabs size="small" items={[
        {
          key: 'info',
          label: <Space><UserOutlined />客户信息</Space>,
          children: customer ? (
            <>
              <Descriptions size="small" column={2} bordered>
                <Descriptions.Item label="姓名">{customer.name}</Descriptions.Item>
                <Descriptions.Item label="电话">{customer.phone}</Descriptions.Item>
                <Descriptions.Item label="公司">{customer.company}</Descriptions.Item>
                <Descriptions.Item label="等级"><Tag color="blue">{customer.level}</Tag></Descriptions.Item>
                <Descriptions.Item label="最近联系">{customer.last_contact}</Descriptions.Item>
                <Descriptions.Item label="标签">
                  {customer.tags?.map((t) => <Tag key={t}>{t}</Tag>)}
                </Descriptions.Item>
              </Descriptions>
              {customer.notes && (
                <div style={{ marginTop: 8, padding: 8, background: '#fafafa', borderRadius: 4, fontSize: 13 }}>
                  {customer.notes}
                </div>
              )}
            </>
          ) : (
            <Empty description="未找到客户信息" image={Empty.PRESENTED_IMAGE_SIMPLE} />
          ),
        },
        {
          key: 'history',
          label: <Space><HistoryOutlined />历史记录</Space>,
          children: customer?.history?.length ? (
            <div style={{ maxHeight: 300, overflowY: 'auto' }}>
              {customer.history.map((h) => (
                <div key={h.id} style={{ padding: '6px 0', borderBottom: '1px solid #f0f0f0' }}>
                  <Space>
                    <Tag>{h.type}</Tag>
                    <span style={{ fontSize: 12, color: '#999' }}>{h.date}</span>
                  </Space>
                  <div style={{ fontSize: 13, marginTop: 4 }}>{h.summary}</div>
                </div>
              ))}
            </div>
          ) : (
            <Empty description="暂无历史" image={Empty.PRESENTED_IMAGE_SIMPLE} />
          ),
        },
        ...(Object.keys(ivrContext).length > 0 ? [{
          key: 'ivr',
          label: <Space><ApiOutlined />IVR 上下文</Space>,
          children: (
            <Descriptions size="small" column={1} bordered>
              {Object.entries(ivrContext).map(([k, v]) => (
                <Descriptions.Item key={k} label={k}>{v}</Descriptions.Item>
              ))}
            </Descriptions>
          ),
        }] : []),
        ...iframeUrls.map((url, idx) => ({
          key: `iframe-${idx}`,
          label: <Space><PhoneOutlined />{iframeUrls.length > 1 ? `业务系统 ${idx + 1}` : '业务系统'}</Space>,
          children: <iframe src={url} style={{ width: '100%', height: 400, border: 'none' }} title={`弹屏-${idx}`} />,
        })),
      ]} />
    </Card>
  );
}
