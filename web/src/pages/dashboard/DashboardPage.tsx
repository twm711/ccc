import { Row, Col, Card, Statistic, Table, Tag, Spin, Alert } from 'antd';
import {
  PhoneOutlined,
  TeamOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  RiseOutlined,
  FallOutlined,
} from '@ant-design/icons';
import { useDashboardOverview, useDashboardAgents, useDashboardFunnel } from '../../api/hooks';

interface Overview {
  total_calls_today: number;
  answered_calls: number;
  abandoned_calls: number;
  active_calls: number;
  agents_online: number;
  agents_busy: number;
  agents_idle: number;
  avg_wait_time_sec: number;
  avg_handle_time_sec: number;
  service_level_20s: number;
  queue_count: number;
  ivr_count: number;
  satisfaction_avg: number;
  callback_pending: number;
  ai_served: number;
  ai_transferred: number;
  total_im_sessions: number;
  im_active: number;
}

interface AgentStatus {
  agent_id: number;
  name: string;
  status: string;
  sub_state: string;
  current_call_duration: number;
  calls_handled: number;
}

interface FunnelData {
  total_inbound: number;
  ivr_completed: number;
  to_agent: number;
  to_bot: number;
  full_service: number;
  half_service: number;
  direct_transfer: number;
  answered: number;
  abandoned: number;
}

export default function DashboardPage() {
  const { data: overview, isLoading: loadingOv, error: errorOv } = useDashboardOverview();
  const { data: agentsData, isLoading: loadingAg } = useDashboardAgents();
  const { data: funnelData, isLoading: loadingFn } = useDashboardFunnel();

  const loading = loadingOv || loadingAg || loadingFn;
  const agents: AgentStatus[] = Array.isArray(agentsData) ? agentsData : [];
  const funnel: FunnelData | null = funnelData ?? null;

  if (loading) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;
  if (errorOv) return <Alert type="error" message="加载仪表盘失败" description={String(errorOv)} />;

  const ov = (overview || {}) as Overview;

  return (
    <>
      <h2>实时概览</h2>
      <Row gutter={[16, 16]}>
        <Col span={4}><Card><Statistic title="今日总呼叫" value={ov.total_calls_today || 0} prefix={<PhoneOutlined />} /></Card></Col>
        <Col span={4}><Card><Statistic title="已接听" value={ov.answered_calls || 0} prefix={<CheckCircleOutlined />} valueStyle={{ color: '#3f8600' }} /></Card></Col>
        <Col span={4}><Card><Statistic title="放弃" value={ov.abandoned_calls || 0} prefix={<WarningOutlined />} valueStyle={{ color: '#cf1322' }} /></Card></Col>
        <Col span={4}><Card><Statistic title="进行中" value={ov.active_calls || 0} prefix={<PhoneOutlined />} valueStyle={{ color: '#1677ff' }} /></Card></Col>
        <Col span={4}><Card><Statistic title="排队中" value={ov.queue_count || 0} prefix={<ClockCircleOutlined />} /></Card></Col>
        <Col span={4}><Card><Statistic title="IVR中" value={ov.ivr_count || 0} prefix={<RiseOutlined />} /></Card></Col>
      </Row>
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col span={4}><Card><Statistic title="在线坐席" value={ov.agents_online || 0} prefix={<TeamOutlined />} /></Card></Col>
        <Col span={4}><Card><Statistic title="忙碌" value={ov.agents_busy || 0} valueStyle={{ color: '#faad14' }} /></Card></Col>
        <Col span={4}><Card><Statistic title="空闲" value={ov.agents_idle || 0} valueStyle={{ color: '#52c41a' }} /></Card></Col>
        <Col span={4}><Card><Statistic title="平均等待(秒)" value={ov.avg_wait_time_sec || 0} prefix={<ClockCircleOutlined />} /></Card></Col>
        <Col span={4}><Card><Statistic title="20s服务水平" value={ov.service_level_20s || 0} suffix="%" prefix={<RiseOutlined />} /></Card></Col>
        <Col span={4}><Card><Statistic title="满意度" value={ov.satisfaction_avg || 0} suffix="/5" prefix={<FallOutlined />} /></Card></Col>
      </Row>

      <Row gutter={16} style={{ marginTop: 16 }}>
        <Col span={16}>
          <Card title="坐席实时状态">
            <Table<AgentStatus>
              dataSource={agents}
              rowKey="agent_id"
              size="small"
              pagination={false}
              columns={[
                { title: '坐席', dataIndex: 'name' },
                { title: '状态', dataIndex: 'status', render: (s: string) => {
                  const colors: Record<string, string> = { READY: 'green', BUSY: 'orange', BREAK: 'blue', OFFLINE: 'default', DIALING: 'purple' };
                  return <Tag color={colors[s] || 'default'}>{s}</Tag>;
                }},
                { title: '子状态', dataIndex: 'sub_state' },
                { title: '当前通话(秒)', dataIndex: 'current_call_duration' },
                { title: '今日处理', dataIndex: 'calls_handled' },
              ]}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card title="呼叫漏斗">
            {funnel && (
              <div>
                {[
                  { label: '总呼入', value: funnel.total_inbound, color: '#1677ff' },
                  { label: 'IVR 完成', value: funnel.ivr_completed, color: '#1890ff' },
                  { label: '转人工', value: funnel.to_agent, color: '#13c2c2' },
                  { label: '转机器人', value: funnel.to_bot, color: '#722ed1' },
                  { label: '全服务', value: funnel.full_service || 0, color: '#2f54eb' },
                  { label: '半服务', value: funnel.half_service || 0, color: '#faad14' },
                  { label: '直转', value: funnel.direct_transfer || 0, color: '#eb2f96' },
                  { label: '已接听', value: funnel.answered, color: '#52c41a' },
                  { label: '放弃', value: funnel.abandoned, color: '#ff4d4f' },
                ].map((step, _i, arr) => {
                  const maxVal = Math.max(...arr.map((a) => a.value), 1);
                  const widthPct = Math.max((step.value / maxVal) * 100, 20);
                  return (
                    <div key={step.label} style={{ marginBottom: 4 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, color: '#666', marginBottom: 2 }}>
                        <span>{step.label}</span>
                        <span style={{ fontWeight: 600 }}>{step.value}</span>
                      </div>
                      <div style={{ width: `${widthPct}%`, height: 20, background: step.color, borderRadius: 4, margin: '0 auto', transition: 'width 0.3s' }} />
                    </div>
                  );
                })}
              </div>
            )}
          </Card>
        </Col>
      </Row>
    </>
  );
}
