import React, { useState, useEffect, useRef } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';
import { Conversation, MessagePayload } from '../types';

// Hardcoded agent ID for demo purposes
const AGENT_ID = 'agent-123';

export const AgentDashboard: React.FC = () => {
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [activeConvId, setActiveConvId] = useState<string | null>(null);
  const [inputText, setInputText] = useState('');
  const [draftText, setDraftText] = useState('');
  
  // Agent connection to hub (listens for all messages broadcasted to agent)
  const wsUrl = `ws://localhost:8080/ws?user_id=${AGENT_ID}&role=agent&conversation_id=all`;
  const { isConnected, messages, setMessages, smartDraft, setSmartDraft, sendMessage, acceptDraft } = useWebSocket(wsUrl);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const draftInputRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    // Fetch queue
    fetch('http://localhost:8080/api/conversations/')
      .then(r => r.json())
      .then(data => setConversations(data || []))
      .catch(console.error);
  }, []);

  useEffect(() => {
    if (activeConvId) {
      // Fetch messages for active conv
      fetch(`http://localhost:8080/api/conversations/${activeConvId}/messages`)
        .then(r => r.json())
        .then(data => setMessages((data || []).reverse()))
        .catch(console.error);
    }
  }, [activeConvId, setMessages]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);
  
  useEffect(() => {
    if (smartDraft) {
      setDraftText(smartDraft.content);
      // Auto focus on draft
      setTimeout(() => draftInputRef.current?.focus(), 100);
    }
  }, [smartDraft]);

  const handleSend = (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputText.trim() || !activeConvId) return;
    sendMessage(inputText, activeConvId);
    setInputText('');
  };

  const activeMessages = messages.filter(m => m.conversation_id === activeConvId);

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
              <button className="text-sm text-slate-500 hover:text-slate-800 transition-colors">Mark Resolved</button>
            </div>
            
            <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-4">
              {activeMessages.map(m => {
                const isAgent = m.sender_id === AGENT_ID;
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
              <div ref={messagesEndRef} />
            </div>

            <div className="p-4 bg-white border-t border-slate-200 flex flex-col gap-2 shrink-0">
              
              {/* AI Smart Draft Box */}
              {smartDraft && smartDraft.conversation_id === activeConvId && (
                <div className="bg-purple-50 border border-purple-200 border-dashed rounded-lg p-4 flex flex-col gap-3 animate-in slide-in-from-bottom-2">
                  <div className="text-purple-600 text-xs font-semibold uppercase tracking-wider flex items-center gap-1">
                    ✨ AI Suggested Reply
                  </div>
                  <textarea 
                    ref={draftInputRef}
                    className="w-full bg-transparent text-slate-800 text-sm focus:outline-none resize-none"
                    value={draftText}
                    onChange={e => setDraftText(e.target.value)}
                    rows={2}
                    onKeyDown={(e) => {
                      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                        // send modified or original draft
                        // Since acceptDraft in backend expects the exact message id,
                        // if they modified it, we should send it as a normal message.
                        if (draftText !== smartDraft.content) {
                          sendMessage(draftText, activeConvId);
                          setSmartDraft(null);
                        } else {
                          acceptDraft(smartDraft.id);
                        }
                      }
                    }}
                  />
                  <div className="flex gap-2">
                    <button 
                      onClick={() => {
                        if (draftText !== smartDraft.content) {
                          sendMessage(draftText, activeConvId);
                          setSmartDraft(null);
                        } else {
                          acceptDraft(smartDraft.id);
                        }
                      }}
                      className="bg-purple-600 hover:bg-purple-700 text-white px-3 py-1.5 rounded-md text-sm font-medium transition-colors"
                    >
                      Accept & Send (Cmd+Enter)
                    </button>
                    <button 
                      onClick={() => setSmartDraft(null)}
                      className="text-slate-500 hover:text-slate-700 px-3 py-1.5 rounded-md text-sm font-medium transition-colors"
                    >
                      Discard
                    </button>
                  </div>
                </div>
              )}

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
