import { useEffect, useRef, useState, useCallback } from 'react';
import type { MessagePayload, WsEvent } from '../types';

export function useWebSocket(url: string | null) {
  const [isConnected, setIsConnected] = useState(false);
  const [messages, setMessages] = useState<MessagePayload[]>([]);
  const [smartDraft, setSmartDraft] = useState<MessagePayload | null>(null);
  const [resolvedConvId, setResolvedConvId] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  const connect = useCallback(() => {
    if (!url) return;

    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      setIsConnected(true);
      console.log('Connected to WS');
    };

    ws.onmessage = (event) => {
      try {
        const data: WsEvent = JSON.parse(event.data);
        if (data.type === 'chat_message') {
          const msg = data.payload as MessagePayload;
          setMessages((prev) => [...prev, msg]);
        } else if (data.type === 'ai_smart_draft') {
          setSmartDraft(data.payload as MessagePayload);
        } else if (data.type === 'SYS_RESOLVE') {
          setResolvedConvId((data as any).conversation_id);
        }
      } catch (e) {
        console.error('Error parsing WS message', e);
      }
    };

    ws.onclose = () => {
      setIsConnected(false);
      console.log('WS Disconnected. Reconnecting in 3s...');
      setTimeout(connect, 3000);
    };

    ws.onerror = (err) => {
      console.error('WS Error:', err);
      ws.close();
    };
  }, [url]);

  useEffect(() => {
    connect();
    return () => {
      if (wsRef.current) {
        // Remove onclose handler to prevent auto-reconnect on intentional component unmount
        wsRef.current.onclose = null;
        wsRef.current.close();
      }
    };
  }, [connect]);

  const sendMessage = useCallback((content: string, conversationId: string) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type: 'chat_message',
        payload: {
          content,
          conversation_id: conversationId,
        }
      }));
    }
  }, []);

  const acceptDraft = useCallback((messageId: string) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type: 'accept_ai_draft',
        payload: {
          message_id: messageId,
        }
      }));
      setSmartDraft(null);
    }
  }, []);

  return { isConnected, messages, setMessages, smartDraft, setSmartDraft, resolvedConvId, setResolvedConvId, sendMessage, acceptDraft };
}
