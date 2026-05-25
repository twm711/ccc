import { useState, useEffect, useCallback } from 'react';
import { Table, Card, Input, Select, DatePicker, Button, Space, Tag, Drawer, Descriptions, Timeline, Empty } from 'antd';
import { SearchOutlined, PlayCircleOutlined, DownloadOutlined, ReloadOutlined } from '@ant-design/icons';
import { callApi, ticketApi } from '../../api/endpoints';
import dayjs from 'dayjs';

interface RelatedTicket {
  id: number;
  subject: string;
  status: string;
  priority: string;
  created_at: string;
}

const typeColors: Record<string, string> = {
  INBOUND: 'green', OUTBOUND: 'blue', INTERNAL: 'purple', BACK2BACK: 'orange',
  CONSULT: 'cyan', MONITOR: 'geekblue', COACH: 'lime', CALLBACK: 'gold',
  PREVIEW: 'magenta', PROGRESSIVE: 'volcano', PREDICTIVE: 'red', POWER: 'blue',
};

interface Call {
  id: number;
  call_type: string;
  caller: string;
  callee: string;
  status: string;
  agent_name: string;
  skill_group_name: string;
  started_at: string;
  ended_at: string;
  duration_sec: number;
  ivr_duration_sec: number;
  queue_duration_sec: number;
  ring_duration_sec: number;
  hangup_cause: string;
  recording_url: string;
  ivr_tracking: { node_id: string; node_type: string; entered_at: string; left_at: string; exit_route: string }[];
  satisfaction_score: number;
  ai_summary: string;
}

export default function CallRecordPage() {
  const [data, setData] = useState<Call[]>([]);
  const [loading, setLoading] = useState(false);
  const [detail, setDetail] = useState<Call | null>(null);
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [relatedTickets, setRelatedTickets] = useState<RelatedTicket[]>([]);

  useEffect(() => {
    if (!detail) { setRelatedTickets([]); return; }
    ticketApi.listByCall(detail.id)
      .then((res) => {
        const items = (res.data as { items?: RelatedTicket[] })?.items || [];
        setRelatedTickets(items);
      })
      .catch(() => setRelatedTickets([]));
  }, [detail]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await callApi.list(filters);
      setData(Array.isArray(res.data) ? res.data : (res.data as Record<string, unknown>)?.items as Call[] || []);
    } catch { /* */ }
    setLoading(false);
  }, [filters]);

  useEffect(() => { load(); }, [load]);

  return (
    <>
      <Card
        title="通话记录"
        extra={
          <Space>
            <Button
              icon={<DownloadOutlined />}
              onClick={() => {
                const qs = new URLSearchParams();
                if (filters.phone) qs.set('caller', filters.phone);
                if (filters.call_type) qs.set('call_type', filters.call_type);
                if (filters.status) qs.set('status', filters.status);
                if (filters.start_date) qs.set('start_from', filters.start_date);
                if (filters.end_date) qs.set('start_to', filters.end_date);
                window.open(`/api/v1/calls/export?${qs.toString()}`, '_blank');
              }}
            >
              导出CSV
            </Button>
            <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
          </Space>
        }
      >
        <Space wrap style={{ marginBottom: 16 }}>
          <Input prefix={<SearchOutlined />} placeholder="搜索号码" onChange={(e) => setFilters({ ...filters, phone: e.target.value })} style={{ width: 180 }} allowClear />
          <Select placeholder="通话类型" allowClear style={{ width: 140 }} onChange={(v) => setFilters({ ...filters, call_type: v })} options={Object.keys(typeColors).map((k) => ({ value: k, label: k }))} />
          <Select placeholder="状态" allowClear style={{ width: 120 }} onChange={(v) => setFilters({ ...filters, status: v })} options={['ringing', 'active', 'ended'].map((v) => ({ value: v, label: v }))} />
          <DatePicker.RangePicker onChange={(_, dates) => setFilters({ ...filters, start_date: dates[0], end_date: dates[1] })} />
        </Space>
        <Table<Call>
          dataSource={data}
          rowKey="id"
          loading={loading}
          size="middle"
          pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 条` }}
          onRow={(record) => ({ onClick: () => setDetail(record), style: { cursor: 'pointer' } })}
          columns={[
            { title: 'ID', dataIndex: 'id', width: 80 },
            { title: '类型', dataIndex: 'call_type', render: (t: string) => <Tag color={typeColors[t]}>{t}</Tag> },
            { title: '主叫', dataIndex: 'caller' },
            { title: '被叫', dataIndex: 'callee' },
            { title: '坐席', dataIndex: 'agent_name' },
            { title: '技能组', dataIndex: 'skill_group_name' },
            { title: '时长(秒)', dataIndex: 'duration_sec' },
            { title: '开始时间', dataIndex: 'started_at', render: (t: string) => t ? dayjs(t).format('YYYY-MM-DD HH:mm:ss') : '-' },
            { title: '挂机原因', dataIndex: 'hangup_cause', render: (c: string) => <Tag>{c}</Tag> },
            { title: '录音', key: 'rec', width: 80, render: (_, r) => r.recording_url ? <PlayCircleOutlined style={{ color: '#1677ff', fontSize: 18 }} /> : '-' },
          ]}
        />
      </Card>
      <Drawer title="通话详情" open={!!detail} onClose={() => setDetail(null)} width={640}>
        {detail && (
          <>
            <Descriptions bordered size="small" column={2}>
              <Descriptions.Item label="通话ID">{detail.id}</Descriptions.Item>
              <Descriptions.Item label="类型"><Tag color={typeColors[detail.call_type]}>{detail.call_type}</Tag></Descriptions.Item>
              <Descriptions.Item label="主叫">{detail.caller}</Descriptions.Item>
              <Descriptions.Item label="被叫">{detail.callee}</Descriptions.Item>
              <Descriptions.Item label="坐席">{detail.agent_name}</Descriptions.Item>
              <Descriptions.Item label="技能组">{detail.skill_group_name}</Descriptions.Item>
              <Descriptions.Item label="状态">{detail.status}</Descriptions.Item>
              <Descriptions.Item label="挂机原因">{detail.hangup_cause}</Descriptions.Item>
              <Descriptions.Item label="总时长">{detail.duration_sec}秒</Descriptions.Item>
              <Descriptions.Item label="IVR时长">{detail.ivr_duration_sec}秒</Descriptions.Item>
              <Descriptions.Item label="排队时长">{detail.queue_duration_sec}秒</Descriptions.Item>
              <Descriptions.Item label="振铃时长">{detail.ring_duration_sec}秒</Descriptions.Item>
              <Descriptions.Item label="满意度">{detail.satisfaction_score || '-'}</Descriptions.Item>
              <Descriptions.Item label="开始">{detail.started_at}</Descriptions.Item>
              <Descriptions.Item label="结束">{detail.ended_at}</Descriptions.Item>
            </Descriptions>
            {detail.recording_url && (
              <Card title="录音" size="small" style={{ marginTop: 16 }}>
                <audio controls src={detail.recording_url} style={{ width: '100%' }} />
                <Button icon={<DownloadOutlined />} href={detail.recording_url} download style={{ marginTop: 8 }}>下载录音</Button>
              </Card>
            )}
            {detail.ai_summary && (
              <Card title="AI 摘要" size="small" style={{ marginTop: 16 }}><p>{detail.ai_summary}</p></Card>
            )}
            {detail.ivr_tracking?.length > 0 && (
              <Card title="IVR 轨迹" size="small" style={{ marginTop: 16 }}>
                <Timeline items={detail.ivr_tracking.map((t) => ({
                  children: `${t.node_type} (${t.node_id}) → ${t.exit_route} [${t.entered_at} ~ ${t.left_at}]`,
                }))} />
              </Card>
            )}
            <Card title="关联工单" size="small" style={{ marginTop: 16 }}>
              {relatedTickets.length === 0 ? (
                <Empty description="无关联工单" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              ) : (
                <Table
                  size="small"
                  rowKey="id"
                  pagination={false}
                  dataSource={relatedTickets}
                  columns={[
                    { title: '工单号', dataIndex: 'id', width: 80 },
                    { title: '主题', dataIndex: 'subject' },
                    { title: '状态', dataIndex: 'status', width: 90, render: (s) => <Tag>{s}</Tag> },
                    { title: '优先级', dataIndex: 'priority', width: 80 },
                    { title: '创建时间', dataIndex: 'created_at', width: 160, render: (t) => t ? dayjs(t).format('MM-DD HH:mm') : '-' },
                  ]}
                />
              )}
            </Card>
          </>
        )}
      </Drawer>
    </>
  );
}
