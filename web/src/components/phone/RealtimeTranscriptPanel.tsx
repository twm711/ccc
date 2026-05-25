import { useState, useEffect, useRef, useCallback } from 'react';
import { Card, Tag, Empty, Switch, Space } from 'antd';
import { AudioOutlined, UserOutlined, RobotOutlined } from '@ant-design/icons';

interface TranscriptLine {
  id: string;
  role: 'agent' | 'customer' | 'system';
  text: string;
  timestamp: string;
  sentiment?: 'positive' | 'neutral' | 'negative';
}

const roleConfig = {
  agent:    { label: '坐席', color: 'blue',    icon: <UserOutlined /> },
  customer: { label: '客户', color: 'green',   icon: <AudioOutlined /> },
  system:   { label: '系统', color: 'default', icon: <RobotOutlined /> },
};

const sentimentColor = { positive: '#52c41a', neutral: '#999', negative: '#ff4d4f' };

export default function RealtimeTranscriptPanel({ callId }: { callId?: number }) {
  const [lines, setLines] = useState<TranscriptLine[]>([]);
  const [autoScroll, setAutoScroll] = useState(true);
  const [connected, setConnected] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);

  const stoppedRef = useRef(false);
  const backoffRef = useRef(1000);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const connectWS = useCallback(() => {
    if (!callId) return;
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws/transcript?call_id=${callId}`;
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      setConnected(true);
      backoffRef.current = 1000;
      setLines([{
        id: 'sys-connected',
        role: 'system',
        text: '实时转写已连接',
        timestamp: new Date().toLocaleTimeString(),
      }]);
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === 'transcript') {
          const line: TranscriptLine = {
            id: data.id || `${Date.now()}-${Math.random()}`,
            role: data.role || 'system',
            text: data.text || '',
            timestamp: data.timestamp || new Date().toLocaleTimeString(),
            sentiment: data.sentiment,
          };
          setLines((prev) => [...prev, line]);
        }
      } catch {
        // ignore malformed messages
      }
    };

    const scheduleReconnect = () => {
      setConnected(false);
      if (stoppedRef.current) return;
      reconnectTimerRef.current = setTimeout(() => {
        backoffRef.current = Math.min(backoffRef.current * 2, 30000);
        connectWS();
      }, backoffRef.current);
    };
    ws.onclose = scheduleReconnect;
    ws.onerror = () => { try { ws.close(); } catch { /* ignore */ } };

    wsRef.current = ws;
  }, [callId]);

  useEffect(() => {
    stoppedRef.current = false;
    connectWS();
    return () => {
      stoppedRef.current = true;
      if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current);
      try { wsRef.current?.close(); } catch { /* ignore */ }
    };
  }, [connectWS]);

  useEffect(() => {
    if (autoScroll) bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [lines, autoScroll]);

  return (
    <Card
      title={
        <Space>
          <AudioOutlined />
          实时转写
          <Tag color={connected ? 'green' : 'red'}>{connected ? '已连接' : '未连接'}</Tag>
        </Space>
      }
      size="small"
      extra={<Switch checkedChildren="自动滚动" unCheckedChildren="暂停" checked={autoScroll} onChange={setAutoScroll} />}
      style={{ height: '100%' }}
      bodyStyle={{ maxHeight: 400, overflowY: 'auto', padding: '8px 12px' }}
    >
      {lines.length === 0 ? (
        <Empty description="等待通话开始..." image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        lines.map((line) => {
          const cfg = roleConfig[line.role];
          return (
            <div key={line.id} style={{ marginBottom: 8, display: 'flex', gap: 8, alignItems: 'flex-start' }}>
              <Tag icon={cfg.icon} color={cfg.color} style={{ flexShrink: 0 }}>{cfg.label}</Tag>
              <div style={{ flex: 1 }}>
                <span style={{ color: line.sentiment ? sentimentColor[line.sentiment] : undefined }}>
                  {line.text}
                </span>
                <span style={{ fontSize: 11, color: '#999', marginLeft: 8 }}>{line.timestamp}</span>
              </div>
            </div>
          );
        })
      )}
      <div ref={bottomRef} />
    </Card>
  );
}
