import { useState, useEffect, useCallback, useRef } from 'react';
import { Badge, Button, Dropdown, Input, Modal, Popover, Select, Space, Tag, Tooltip, message } from 'antd';
import {
  PhoneOutlined, PhoneFilled, PauseCircleOutlined, SwapOutlined,
  TeamOutlined, AudioOutlined, AudioMutedOutlined, ClockCircleOutlined,
  CoffeeOutlined, CheckCircleOutlined, PoweroffOutlined,
  NumberOutlined, FormOutlined,
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
  const [acwCallId, setAcwCallId] = useState<number | null>(null);
  const callIdRef = useRef<number | null>(null);

  // Keep ref in sync with state so WebSocket handler can read current callId
  useEffect(() => { callIdRef.current = callId; }, [callId]);

  // WebSocket for real-time call events with auto-reconnect + heartbeat.
  // Without these, a single network hiccup makes the agent unreachable until
  // they refresh — a P0 prod incident in a 24/7 call center.
  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws/agent-events`;
    let ws: WebSocket | null = null;
    let heartbeat: ReturnType<typeof setInterval> | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let backoffMs = 1000;
    let stopped = false;

    const connect = () => {
      ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        backoffMs = 1000;
        heartbeat = setInterval(() => {
          if (ws?.readyState === WebSocket.OPEN) {
            try { ws.send(JSON.stringify({ type: 'ping' })); } catch { /* ignore */ }
          }
        }, 30000);
      };

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
              setAcwCallId(callIdRef.current);
              setStatus('acw');
              setHeld(false);
              setMuted(false);
              break;
          }
        } catch {
          // ignore
        }
      };

      const scheduleReconnect = () => {
        if (stopped) return;
        if (heartbeat) { clearInterval(heartbeat); heartbeat = null; }
        reconnectTimer = setTimeout(connect, backoffMs);
        backoffMs = Math.min(backoffMs * 2, 30000);
      };
      ws.onclose = scheduleReconnect;
      ws.onerror = () => { try { ws?.close(); } catch { /* ignore */ } };
    };

    connect();

    return () => {
      stopped = true;
      if (heartbeat) clearInterval(heartbeat);
      if (reconnectTimer) clearTimeout(reconnectTimer);
      try { ws?.close(); } catch { /* ignore */ }
    };
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
    if (!callId) return;
    try {
      await callControlApi.answer(callId);
      setStatus('talking');
      setDuration(0);
      message.success('已接听');
    } catch {
      message.error('接听失败');
    }
  };

  const handleHangup = async () => {
    if (!callId) return;
    try {
      await callControlApi.end(callId);
      setStatus('acw');
      setCallId(null);
      setDuration(0);
      setHeld(false);
      setMuted(false);
      message.info('通话结束');
    } catch {
      message.error('挂机失败');
    }
  };

  const handleSendDTMF = async (digit: string) => {
    if (!callId) return;
    try {
      await callControlApi.sendDTMF(callId, { digits: digit });
    } catch {
      message.error('DTMF发送失败');
    }
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

  const [dispositionOpen, setDispositionOpen] = useState(false);
  const [dispositionCode, setDispositionCode] = useState('');
  const [dispositionNote, setDispositionNote] = useState('');

  const handleFinishAcw = async () => {
    if (acwCallId && dispositionCode) {
      try {
        await callControlApi.disposition(acwCallId, { disposition_code: dispositionCode, note: dispositionNote });
      } catch { /* best effort */ }
    }
    await changeStatus('idle');
    setDispositionCode('');
    setDispositionNote('');
    setAcwCallId(null);
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
            <Popover
              trigger="click"
              content={
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 48px)', gap: 4 }}>
                  {['1','2','3','4','5','6','7','8','9','*','0','#'].map((d) => (
                    <Button key={d} size="small" onClick={() => handleSendDTMF(d)}>{d}</Button>
                  ))}
                </div>
              }
            >
              <Tooltip title="DTMF">
                <Button icon={<NumberOutlined />} />
              </Tooltip>
            </Popover>
            <Tooltip title="转接">
              <Button icon={<SwapOutlined />} onClick={() => setTransferOpen(true)} />
            </Tooltip>
            <Tooltip title="会议">
              <Button icon={<TeamOutlined />} onClick={handleConference} />
            </Tooltip>
            <Button danger icon={<PhoneFilled />} onClick={() => { setAcwCallId(callId); handleHangup(); }}>挂机</Button>
          </Space>
        )}

        {status === 'acw' && (
          <Space>
            <Button icon={<FormOutlined />} onClick={() => setDispositionOpen(true)}>标记结果</Button>
            <Button type="primary" onClick={handleFinishAcw}>完成话后处理</Button>
          </Space>
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

      <Modal
        title="通话结果标记"
        open={dispositionOpen}
        onCancel={() => setDispositionOpen(false)}
        onOk={() => setDispositionOpen(false)}
        okText="确定"
      >
        <div style={{ marginBottom: 12 }}>
          <label>结果代码</label>
          <Select
            value={dispositionCode || undefined}
            onChange={setDispositionCode}
            placeholder="选择通话结果"
            style={{ width: '100%', marginTop: 4 }}
            options={[
              { value: 'resolved', label: '已解决' },
              { value: 'follow_up', label: '需跟进' },
              { value: 'escalated', label: '已升级' },
              { value: 'no_answer', label: '未接听' },
              { value: 'voicemail', label: '留言' },
              { value: 'wrong_number', label: '错号' },
              { value: 'callback', label: '需回拨' },
            ]}
          />
        </div>
        <div>
          <label>备注</label>
          <Input.TextArea
            value={dispositionNote}
            onChange={(e) => setDispositionNote(e.target.value)}
            rows={3}
            placeholder="通话备注"
            style={{ marginTop: 4 }}
          />
        </div>
      </Modal>
    </>
  );
}
