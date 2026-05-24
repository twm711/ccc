import { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, Progress, Table, Tag, Space, Button, Select } from 'antd';
import {
  PhoneOutlined, TeamOutlined, CheckCircleOutlined, CloseCircleOutlined,
  ClockCircleOutlined, ReloadOutlined, DashboardOutlined, ThunderboltOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { campaignApi } from '../../api/endpoints';

interface CampaignStats {
  total_cases: number;
  completed: number;
  connected: number;
  failed: number;
  pending: number;
  in_progress: number;
  connect_rate: number;
  abandon_rate: number;
  avg_duration: number;
  concurrent: number;
  agents_active: number;
  agents_idle: number;
  elapsed_min: number;
}

interface AgentStatus {
  id: number;
  name: string;
  status: string;
  current_case: string;
  calls_made: number;
  connected: number;
}

const primaryGradient = { '0%': '#108ee9', '100%': '#87d068' };

export default function CampaignLiveDashboard() {
  const [campaigns, setCampaigns] = useState<{ id: number; name: string; status: string }[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [stats, setStats] = useState<CampaignStats | null>(null);
  const [agents, setAgents] = useState<AgentStatus[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    campaignApi.list().then((res) => {
      const items = Array.isArray(res.data) ? res.data : res.data?.items || [];
      const running = items.filter((c: Record<string, unknown>) => c.status === 'running' || c.status === 'paused');
      setCampaigns(running);
      if (running.length > 0 && !selectedId) setSelectedId(running[0].id);
    }).catch(() => {});
  }, []);

  const loadStats = async () => {
    if (!selectedId) return;
    setLoading(true);
    try {
      const res = await campaignApi.stats(selectedId);
      setStats(res.data?.summary || res.data);
      setAgents(res.data?.agents || []);
    } catch { /* ignore */ }
    setLoading(false);
  };

  useEffect(() => {
    loadStats();
    const t = setInterval(loadStats, 5000);
    return () => clearInterval(t);
  }, [selectedId]);

  const agentColumns: ColumnsType<AgentStatus> = [
    { title: '坐席', dataIndex: 'name', width: 100 },
    {
      title: '状态', dataIndex: 'status', width: 80,
      render: (v) => <Tag color={v === 'talking' ? 'blue' : v === 'idle' ? 'green' : 'default'}>{v}</Tag>,
    },
    { title: '当前案例', dataIndex: 'current_case', width: 120, ellipsis: true },
    { title: '已拨', dataIndex: 'calls_made', width: 60 },
    { title: '接通', dataIndex: 'connected', width: 60 },
  ];

  const progress = stats ? Math.round((stats.completed / Math.max(stats.total_cases, 1)) * 100) : 0;

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}><DashboardOutlined /> 外呼实时大屏</h2>
        <Space>
          <Select
            value={selectedId ?? undefined}
            onChange={setSelectedId}
            style={{ width: 200 }}
            placeholder="选择活动"
            options={campaigns.map((c) => ({ value: c.id, label: `${c.name} (${c.status})` }))}
          />
          <Button icon={<ReloadOutlined />} onClick={loadStats} loading={loading}>刷新</Button>
        </Space>
      </div>

      {stats && (
        <>
          {/* KPI cards */}
          <Row gutter={[16, 16]}>
            <Col span={4}><Card><Statistic title="总案例" value={stats.total_cases} /></Card></Col>
            <Col span={4}><Card><Statistic title="已完成" value={stats.completed} prefix={<CheckCircleOutlined />} valueStyle={{ color: '#52c41a' }} /></Card></Col>
            <Col span={4}><Card><Statistic title="接通数" value={stats.connected} prefix={<PhoneOutlined />} valueStyle={{ color: '#1677ff' }} /></Card></Col>
            <Col span={4}><Card><Statistic title="失败" value={stats.failed} prefix={<CloseCircleOutlined />} valueStyle={{ color: '#ff4d4f' }} /></Card></Col>
            <Col span={4}><Card><Statistic title="并发数" value={stats.concurrent} prefix={<ThunderboltOutlined />} valueStyle={{ color: '#fa8c16' }} /></Card></Col>
            <Col span={4}><Card><Statistic title="用时(分)" value={stats.elapsed_min} prefix={<ClockCircleOutlined />} /></Card></Col>
          </Row>

          {/* Progress and rates */}
          <Row gutter={16} style={{ marginTop: 16 }}>
            <Col span={8}>
              <Card title="完成进度">
                <Progress type="dashboard" percent={progress} size={160} strokeColor={primaryGradient} />
                <div style={{ textAlign: 'center', marginTop: 8, color: '#666' }}>
                  {stats.completed} / {stats.total_cases}
                </div>
              </Card>
            </Col>
            <Col span={8}>
              <Card title="接通率 / 放弃率">
                <Row gutter={16}>
                  <Col span={12}>
                    <Progress type="circle" percent={Math.round(stats.connect_rate * 100)} size={120} strokeColor="#52c41a" format={(p) => `${p}%`} />
                    <div style={{ textAlign: 'center', marginTop: 4 }}>接通率</div>
                  </Col>
                  <Col span={12}>
                    <Progress type="circle" percent={Math.round(stats.abandon_rate * 100)} size={120} strokeColor="#ff4d4f" format={(p) => `${p}%`} />
                    <div style={{ textAlign: 'center', marginTop: 4 }}>放弃率</div>
                  </Col>
                </Row>
              </Card>
            </Col>
            <Col span={8}>
              <Card title="坐席概况">
                <Row gutter={16}>
                  <Col span={12}>
                    <Statistic title="活跃坐席" value={stats.agents_active} prefix={<TeamOutlined />} valueStyle={{ color: '#1677ff' }} />
                  </Col>
                  <Col span={12}>
                    <Statistic title="空闲坐席" value={stats.agents_idle} prefix={<TeamOutlined />} valueStyle={{ color: '#52c41a' }} />
                  </Col>
                </Row>
                <Statistic title="平均通话(秒)" value={stats.avg_duration} prefix={<ClockCircleOutlined />} style={{ marginTop: 16 }} />
              </Card>
            </Col>
          </Row>

          {/* Agent table */}
          <Card title="坐席明细" style={{ marginTop: 16 }}>
            <Table<AgentStatus> columns={agentColumns} dataSource={agents} rowKey="id" size="small" pagination={false} />
          </Card>
        </>
      )}
    </>
  );
}
