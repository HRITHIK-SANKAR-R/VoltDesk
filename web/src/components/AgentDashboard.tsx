import React, { useState, useEffect } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';
import type { Conversation } from '../types';

interface AgentDashboardProps {
  userId: string;
}

export const AgentDashboard: React.FC<AgentDashboardProps> = ({ userId }) => {
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [activeConvId, setActiveConvId] = useState<string | null>(null);
  const [inputText, setInputText] = useState('');
  
  // Agent connection to hub (listens for all messages broadcasted to agent)
  const wsUrl = `ws://${window.location.host}/ws?user_id=${userId}&role=agent&conversation_id=all`;
  const { isConnected, messages, setMessages, smartDraft, setSmartDraft, resolvedConvId, setResolvedConvId, sendMessage } = useWebSocket(wsUrl);

  useEffect(() => {
    // Fetch queue
    fetch('/api/conversations/')
      .then(r => r.json())
      .then(data => setConversations(data || []))
      .catch(console.error);
  }, []);

  useEffect(() => {
    if (activeConvId) {
      // Fetch messages for active conv
      fetch(`/api/conversations/${activeConvId}/messages`)
        .then(r => r.json())
        .then(data => setMessages((data || []).reverse()))
        .catch(console.error);
    }
  }, [activeConvId, setMessages]);
  
  useEffect(() => {
    if (resolvedConvId) {
      setConversations(prev => prev.filter(c => c.id !== resolvedConvId));
      if (activeConvId === resolvedConvId) {
        setActiveConvId(null);
        setInputText('');
      }
      setResolvedConvId(null);
    }
  }, [resolvedConvId, activeConvId, setResolvedConvId]);

  useEffect(() => {
    if (smartDraft && smartDraft.conversation_id === activeConvId) {
      if (!inputText.trim()) {
        setInputText(smartDraft.content);
      }
      setSmartDraft(null); // Move directly to input
    }
  }, [smartDraft, activeConvId, inputText, setSmartDraft]);

  const handleResolve = async () => {
    if (!activeConvId) return;
    try {
      const res = await fetch(`/api/conversations/${activeConvId}/resolve`, { method: 'PATCH' });
      if (res.ok) {
        setConversations(prev => prev.filter(c => c.id !== activeConvId));
        setActiveConvId(null);
        setInputText('');
        setSmartDraft(null);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handleSend = (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputText.trim() || !activeConvId) return;
    sendMessage(inputText, activeConvId);
    setInputText('');
  };

  const activeMessages = messages.filter(m => m.conversation_id === activeConvId);
  const reversedMessages = [...activeMessages].reverse();

  return (
    <div className="w-screen h-screen overflow-hidden flex bg-slate-50 text-slate-900">
      {/* Left Sidebar Queue */}
      <div className="w-80 flex-shrink-0 bg-white border-r border-slate-200 flex flex-col z-10">
        <div className="p-4 border-b border-slate-200 shrink-0">
          <h2 className="font-semibold text-lg">Active Queue</h2>
          <div className="text-sm text-slate-500 flex items-center gap-2 mt-1">
            <span className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`}></span>
            {isConnected ? 'Online' : 'Disconnected'}
          </div>
        </div>
        
        <div className="flex-1 overflow-y-auto">
          {conversations.map(conv => (
            <div 
              key={conv.id}
              onClick={() => setActiveConvId(conv.id)}
              className={`p-4 border-b border-slate-100 cursor-pointer transition-colors ${activeConvId === conv.id ? 'bg-orange-50 border-l-4 border-l-orange-600' : 'bg-white hover:bg-slate-50'}`}
            >
              <div className="flex justify-between items-start mb-1">
                <span className="font-medium text-sm truncate pr-2">Customer {conv.customer_id.substring(0,6)}</span>
                <span className="text-xs text-slate-400 shrink-0">
                  {new Date(conv.last_activity_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                </span>
              </div>
              <div className="text-xs text-slate-500 truncate">
                Status: {conv.status}
              </div>
            </div>
          ))}
          {conversations.length === 0 && (
             <div className="p-6 text-center text-slate-400 text-sm">No active conversations.</div>
          )}
        </div>
      </div>

      {/* Main Conversation Window */}
      <div className="flex-1 flex flex-col relative min-w-0">
        {activeConvId ? (
          <>
            <div className="h-14 bg-white border-b border-slate-200 flex items-center px-6 justify-between shrink-0 shadow-sm z-10">
              <h3 className="font-semibold">Conversation ID: {activeConvId.substring(0,8)}...</h3>
              <button onClick={handleResolve} className="text-sm text-slate-500 hover:text-slate-800 transition-colors">Mark Resolved</button>
            </div>
            
            <div className="flex-1 overflow-y-auto flex flex-col-reverse p-6 gap-4">
              {reversedMessages.map(m => {
                const isAgent = m.sender_id === userId || m.sender_id === '00000000-0000-0000-0000-000000000000';
                return (
                  <div key={m.id} className={`flex ${isAgent ? 'justify-end' : 'justify-start'}`}>
                    <div className={`max-w-[70%] px-5 py-3 text-sm shadow-sm ${
                      isAgent 
                        ? 'bg-slate-800 text-white rounded-l-xl rounded-tr-xl' 
                        : 'bg-white border border-slate-200 text-slate-800 rounded-r-xl rounded-tl-xl'
                    }`}>
                      {m.content}
                    </div>
                  </div>
                );
              })}
            </div>

            <div className="p-4 bg-white border-t border-slate-200 flex flex-col gap-2 shrink-0">
              
              {/* Standard Input */}
              <form onSubmit={handleSend} className="flex gap-2 items-center">
                <input 
                  type="text"
                  placeholder="Type your reply..."
                  className="flex-1 bg-slate-100 rounded-lg px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-slate-300"
                  value={inputText}
                  onChange={e => setInputText(e.target.value)}
                />
                <button type="submit" disabled={!inputText.trim()} className="bg-slate-800 hover:bg-slate-900 text-white px-6 py-3 rounded-lg font-medium text-sm transition-colors disabled:opacity-50">
                  Send
                </button>
              </form>
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-slate-400">
            Select a conversation from the queue
          </div>
        )}
      </div>
    </div>
  );
};
