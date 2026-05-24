import { useState, useEffect, useCallback } from 'react';
import { Badge, Button, Dropdown, Space, Tag, Tooltip, message } from 'antd';
import {
  PhoneOutlined, PhoneFilled, PauseCircleOutlined, SwapOutlined,
  TeamOutlined, AudioOutlined, AudioMutedOutlined, ClockCircleOutlined,
  CoffeeOutlined, CheckCircleOutlined, PoweroffOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { callControlApi, agentPresenceApi } from '../../api/endpoints';
import TransferModal from './TransferModal';

type AgentStatus = 'idle' | 'ringing' | 'talking' | 'acw' | 'break' | 'offline';

const statusConfig: Record<AgentStatus, { color: string; label: string; icon: React.ReactNode }> = {
  idle:    { color: 'green',   label: '空闲',   icon: <CheckCircleOutlined /> },
  ringing: { color: 'orange',  label: '振铃',   icon: <PhoneOutlined /> },
  talking: { color: 'blue',    label: '通话中', icon: <PhoneFilled /> },
  acw:     { color: 'purple',  label: '话后处理', icon: <ClockCircleOutlined /> },
  break:   { color: 'gold',    label: '小休',   icon: <CoffeeOutlined /> },
  offline: { color: 'default', label: '离线',   icon: <PoweroffOutlined /> },
};

const breakReasons = [
  { value: 'lunch', label: '午餐' },
  { value: 'rest', label: '休息' },
  { value: 'meeting', label: '会议' },
  { value: 'training', label: '培训' },
  { value: 'personal', label: '私事' },
];

export default function AgentPhoneBar() {
  const [status, setStatus] = useState<AgentStatus>('offline');
  const [callId, setCallId] = useState<number | null>(null);
  const [muted, setMuted] = useState(false);
  const [held, setHeld] = useState(false);
  const [duration, setDuration] = useState(0);
  const [transferOpen, setTransferOpen] = useState(false);

  // WebSocket for real-time call events
  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws/agent-events`;
    const ws = new WebSocket(wsUrl);

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        switch (data.type) {
          case 'call_ringing':
            setStatus('ringing');
            setCallId(data.call_id);
            break;
          case 'call_answered':
            setStatus('talking');
            setDuration(0);
            break;
          case 'call_ended':
            setStatus('acw');
            setHeld(false);
            setMuted(false);
            break;
        }
      } catch {
        // ignore
      }
    };

    return () => ws.close();
  }, []);

  // Timer for call duration.
  useEffect(() => {
    if (status !== 'talking') return;
    const timer = setInterval(() => setDuration((d) => d + 1), 1000);
    return () => clearInterval(timer);
  }, [status]);

  const formatDuration = (s: number) => {
    const m = Math.floor(s / 60);
    const sec = s % 60;
    return `${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}`;
  };

  const changeStatus = useCallback(async (newStatus: string, reason?: string) => {
    try {
      await agentPresenceApi.changeStatus({ status: newStatus, ...(reason && { reason }) });
      setStatus(newStatus as AgentStatus);
    } catch {
      message.error('状态切换失败');
    }
  }, []);

  const handleAnswer = async () => {
    setStatus('talking');
    setDuration(0);
    message.success('已接听');
  };

  const handleHangup = async () => {
    setStatus('acw');
    setCallId(null);
    setDuration(0);
    setHeld(false);
    setMuted(false);
    message.info('通话结束');
  };

  const handleHold = async () => {
    if (!callId) return;
    try {
      if (held) {
        await callControlApi.retrieve(callId);
        setHeld(false);
      } else {
        await callControlApi.hold(callId);
        setHeld(true);
      }
    } catch {
      message.error('操作失败');
    }
  };

  const handleConference = async () => {
    if (!callId) return;
    try {
      await callControlApi.conference(callId);
      message.success('已发起会议');
    } catch {
      message.error('会议发起失败');
    }
  };

  const handleFinishAcw = () => {
    setStatus('idle');
    message.success('话后处理完成');
  };

  const breakMenu: MenuProps['items'] = breakReasons.map((r) => ({
    key: r.value,
    label: r.label,
    onClick: () => changeStatus('break', r.value),
  }));

  const cfg = statusConfig[status];

  return (
    <>
      <div style={{
        display: 'flex', alignItems: 'center', gap: 12, padding: '8px 16px',
        background: '#fff', borderBottom: '1px solid #f0f0f0', boxShadow: '0 1px 4px rgba(0,0,0,0.06)',
      }}>
        {/* Status indicator */}
        <Badge status={cfg.color as 'success' | 'warning' | 'processing' | 'default' | 'error'} />
        <Tag icon={cfg.icon} color={cfg.color}>{cfg.label}</Tag>

        {/* Call duration */}
        {status === 'talking' && (
          <Tag icon={<ClockCircleOutlined />} color="blue">{formatDuration(duration)}</Tag>
        )}

        <div style={{ flex: 1 }} />

        {/* Call control buttons */}
        {status === 'ringing' && (
          <Button type="primary" icon={<PhoneOutlined />} onClick={handleAnswer} style={{ background: '#52c41a' }}>
            接听
          </Button>
        )}

        {status === 'talking' && (
          <Space>
            <Tooltip title={muted ? '取消静音' : '静音'}>
              <Button
                icon={muted ? <AudioMutedOutlined /> : <AudioOutlined />}
                onClick={() => setMuted(!muted)}
                type={muted ? 'primary' : 'default'}
                danger={muted}
              />
            </Tooltip>
            <Tooltip title={held ? '恢复' : '保持'}>
              <Button
                icon={<PauseCircleOutlined />}
                onClick={handleHold}
                type={held ? 'primary' : 'default'}
              />
            </Tooltip>
            <Tooltip title="转接">
              <Button icon={<SwapOutlined />} onClick={() => setTransferOpen(true)} />
            </Tooltip>
            <Tooltip title="会议">
              <Button icon={<TeamOutlined />} onClick={handleConference} />
            </Tooltip>
            <Button danger icon={<PhoneFilled />} onClick={handleHangup}>挂机</Button>
          </Space>
        )}

        {status === 'acw' && (
          <Button type="primary" onClick={handleFinishAcw}>完成话后处理</Button>
        )}

        {/* Status controls */}
        {(status === 'idle' || status === 'break' || status === 'offline') && (
          <Space>
            {status === 'offline' && (
              <Button type="primary" onClick={() => changeStatus('idle')}>上线</Button>
            )}
            {status === 'idle' && (
              <Dropdown menu={{ items: breakMenu }}>
                <Button icon={<CoffeeOutlined />}>小休</Button>
              </Dropdown>
            )}
            {status === 'break' && (
              <Button type="primary" onClick={() => changeStatus('idle')}>恢复空闲</Button>
            )}
            {status !== 'offline' && (
              <Button icon={<PoweroffOutlined />} onClick={() => changeStatus('offline')}>下线</Button>
            )}
          </Space>
        )}
      </div>

      <TransferModal
        open={transferOpen}
        callId={callId}
        onClose={() => setTransferOpen(false)}
      />
    </>
  );
}
