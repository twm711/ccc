import { useState } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu, theme, Dropdown, Avatar, Space } from 'antd';
import {
  DashboardOutlined,
  TeamOutlined,
  PhoneOutlined,
  ApartmentOutlined,
  SettingOutlined,
  BarChartOutlined,
  CustomerServiceOutlined,
  FileTextOutlined,
  MessageOutlined,
  RobotOutlined,
  SoundOutlined,
  UserOutlined,
  LogoutOutlined,
  NumberOutlined,
  TagOutlined,
  MailOutlined,
  AuditOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { useAuthStore } from '../../stores/auth';
import AgentPhoneBar from '../phone/AgentPhoneBar';

const { Header, Sider, Content } = Layout;

const menuItems: MenuProps['items'] = [
  { key: '/dashboard', icon: <DashboardOutlined />, label: '概览' },
  {
    key: 'agent-mgmt',
    icon: <TeamOutlined />,
    label: '客服管理',
    children: [
      { key: '/agents', label: '坐席列表' },
      { key: '/skill-groups', label: '技能组' },
    ],
  },
  {
    key: 'flow-mgmt',
    icon: <ApartmentOutlined />,
    label: '流程管理',
    children: [
      { key: '/ivr', label: 'IVR 流程' },
      { key: '/audio-files', label: '音频管理' },
    ],
  },
  {
    key: 'telephony',
    icon: <PhoneOutlined />,
    label: '话务管理',
    children: [
      { key: '/phone-numbers', label: '号码管理' },
      { key: '/call-records', label: '通话记录' },
      { key: '/voicemails', label: '语音信箱' },
    ],
  },
  {
    key: 'campaigns',
    icon: <SoundOutlined />,
    label: '批量外呼',
    children: [
      { key: '/campaigns', label: '外呼活动' },
      { key: '/campaigns/live', label: '实时大屏' },
    ],
  },
  {
    key: 'im',
    icon: <MessageOutlined />,
    label: '在线客服',
    children: [
      { key: '/im/sessions', label: '会话管理' },
      { key: '/im/channels', label: '渠道配置' },
      { key: '/im/social', label: '社交渠道' },
    ],
  },
  {
    key: 'business',
    icon: <CustomerServiceOutlined />,
    label: '业务管理',
    children: [
      { key: '/crm/customers', label: '客户管理' },
      { key: '/tickets', label: '工单管理' },
      { key: '/ticket-templates', label: '工单模板' },
      { key: '/knowledge', label: '知识库' },
    ],
  },
  {
    key: 'reports',
    icon: <BarChartOutlined />,
    label: '数据报表',
    children: [
      { key: '/reports/agents', label: '坐席报表' },
      { key: '/reports/group-agents', label: '分组报表' },
      { key: '/reports/skill-groups', label: '技能组报表' },
      { key: '/reports/b2b', label: '双呼报表' },
      { key: '/reports/internal', label: '内部呼叫' },
      { key: '/reports/status-log', label: '状态日志' },
      { key: '/reports/csat', label: '满意度' },
      { key: '/reports/campaigns', label: '活动报表' },
    ],
  },
  {
    key: 'ai',
    icon: <RobotOutlined />,
    label: '智能化',
    children: [
      { key: '/ai/digital-employees', label: '数字员工' },
      { key: '/ai/qa', label: '智能质检' },
      { key: '/ai/hotwords', label: 'ASR 热词' },
      { key: '/ai/performance', label: '绩效管理' },
      { key: '/ai/annotations', label: '标注管理' },
      { key: '/ai/llm-models', label: 'LLM 网关' },
      { key: '/ai/advanced', label: '高级 AI' },
    ],
  },
  {
    key: 'settings',
    icon: <SettingOutlined />,
    label: '设置',
    children: [
      { key: '/settings/tenant', label: '租户设置' },
      { key: '/settings/break-reasons', label: '小休原因' },
      { key: '/settings/disposition-codes', label: '结案代码' },
      { key: '/settings/business-hours', label: '营业时间' },
      { key: '/settings/custom-fields', label: '自定义字段' },
      { key: '/settings/call-tags', icon: <TagOutlined />, label: '号码标签' },
      { key: '/settings/screen-pop', label: '来电弹屏' },
      { key: '/settings/webhooks', label: '事件推送' },
      { key: '/settings/sms', icon: <MailOutlined />, label: '短信配置' },
      { key: '/settings/quick-replies', label: '快捷回复' },
      { key: '/settings/csat', label: '满意度配置' },
    ],
  },
  {
    key: 'platform',
    icon: <AuditOutlined />,
    label: '平台管理',
    children: [
      { key: '/tenants', icon: <NumberOutlined />, label: '实例管理' },
      { key: '/audit-logs', icon: <FileTextOutlined />, label: '审计日志' },
    ],
  },
];

export default function AppLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { token: { colorBgContainer, borderRadiusLG } } = theme.useToken();
  const { user, logout } = useAuthStore();

  const userMenu: MenuProps['items'] = [
    { key: 'profile', icon: <UserOutlined />, label: '个人中心', onClick: () => navigate('/me') },
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: logout },
  ];

  const selectedKeys = [location.pathname];
  const openKeys = menuItems
    ?.filter((item): item is { key: string; children: MenuProps['items'] } =>
      !!(item && 'children' in item))
    .filter((item) => item.children?.some((c) => c && 'key' in c && location.pathname.startsWith(c.key as string)))
    .map((item) => item.key) || [];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider collapsible collapsed={collapsed} onCollapse={setCollapsed} width={220}>
        <div style={{ height: 40, margin: 16, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <span style={{ color: '#fff', fontWeight: 700, fontSize: collapsed ? 16 : 20, whiteSpace: 'nowrap' }}>
            {collapsed ? 'CCC' : 'CCC 联络中心'}
          </span>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={selectedKeys}
          defaultOpenKeys={openKeys}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <AgentPhoneBar />
        <Header style={{ padding: '0 24px', background: colorBgContainer, display: 'flex', justifyContent: 'flex-end', alignItems: 'center' }}>
          <Dropdown menu={{ items: userMenu }} placement="bottomRight">
            <Space style={{ cursor: 'pointer' }}>
              <Avatar icon={<UserOutlined />} />
              <span>{user?.username || 'Admin'}</span>
            </Space>
          </Dropdown>
        </Header>
        <Content style={{ margin: 16 }}>
          <div style={{ padding: 24, minHeight: 360, background: colorBgContainer, borderRadius: borderRadiusLG }}>
            <Outlet />
          </div>
        </Content>
      </Layout>
    </Layout>
  );
}
