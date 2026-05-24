import { useState } from 'react';
import { Card, Table, DatePicker, Button, Space, Tag } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { reportApi } from '../../api/endpoints';

export default function CampaignReportPage() {
  const [data, setData] = useState<Record<string, unknown>[]>([]);
  const [loading, setLoading] = useState(false);
  const [params, setParams] = useState<Record<string, string>>({});

  const load = async () => {
    setLoading(true);
    try { const res = await reportApi.campaigns(params); setData(Array.isArray(res.data) ? res.data : []); } catch { /* */ }
    setLoading(false);
  };

  return (
    <Card title="活动报表">
      <Space style={{ marginBottom: 16 }}>
        <DatePicker.RangePicker onChange={(_, dates) => setParams({ ...params, start_date: dates[0], end_date: dates[1] })} />
        <Button icon={<ReloadOutlined />} onClick={load}>查询</Button>
      </Space>
      <Table dataSource={data} rowKey="id" loading={loading} size="small" columns={[
        { title: '活动名称', dataIndex: 'campaign_name' },
        { title: '模式', dataIndex: 'dialing_mode', render: (v: string) => <Tag color="blue">{v}</Tag> },
        { title: '总案例', dataIndex: 'total_cases' },
        { title: '已完成', dataIndex: 'completed_cases' },
        { title: '接通数', dataIndex: 'connected' },
        { title: '接通率', dataIndex: 'connect_rate', render: (v: number) => `${((v || 0) * 100).toFixed(1)}%` },
        { title: '平均时长(秒)', dataIndex: 'avg_duration' },
        { title: '放弃率', dataIndex: 'abandon_rate', render: (v: number) => `${((v || 0) * 100).toFixed(1)}%` },
      ]} />
    </Card>
  );
}
