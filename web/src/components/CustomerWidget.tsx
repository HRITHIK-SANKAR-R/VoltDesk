import React, { useState, useEffect } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';

export const CustomerWidget: React.FC = () => {
  const [isOpen, setIsOpen] = useState(false);
  const [inputText, setInputText] = useState('');
  const [email, setEmail] = useState('');
  const [session, setSession] = useState<{ user_id: string; conversation_id: string } | null>(null);
  
  const wsUrl = session ? `ws://localhost:8081/ws?user_id=${session.user_id}&role=customer&conversation_id=${session.conversation_id}` : null;
  const { isConnected, messages, setMessages, sendMessage } = useWebSocket(wsUrl);

  const handleStartSession = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email) return;
    try {
      const res = await fetch(`http://localhost:8081/api/auth/customer?email=${encodeURIComponent(email)}`, { method: 'POST' });
      const data = await res.json();
      setSession(data);
      
      const histRes = await fetch(`http://localhost:8081/api/conversations/${data.conversation_id}/messages`);
      const hist = await histRes.json();
      if (hist) setMessages(hist.reverse());
    } catch (err) {
      console.error(err);
    }
  };

  const handleSend = (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputText.trim() || !session) return;
    sendMessage(inputText, session.conversation_id);
    setInputText('');
  };

  if (!isOpen) {
    return (
      <button 
        onClick={() => setIsOpen(true)}
        className="fixed bottom-6 right-6 w-14 h-14 rounded-full bg-orange-600 shadow-lg hover:scale-105 transition-transform flex items-center justify-center text-white z-50"
      >
        <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
        </svg>
      </button>
    );
  }

  // Reverse messages for flex-col-reverse
  const reversedMessages = [...messages].reverse();

  return (
    <div className="fixed bottom-0 right-0 sm:bottom-6 sm:right-6 w-full h-full sm:w-80 sm:h-96 bg-white sm:rounded-xl shadow-2xl flex flex-col overflow-hidden z-50 border border-slate-200">
      <div className="bg-orange-600 text-white p-4 flex justify-between items-center shrink-0">
        <h3 className="font-semibold">Support</h3>
        <button onClick={() => setIsOpen(false)} className="text-white hover:bg-orange-700 p-1 rounded transition-colors">
          <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
             <path fillRule="evenodd" d="M3 10a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
          </svg>
        </button>
      </div>

      {!session ? (
        <form onSubmit={handleStartSession} className="p-4 flex flex-col gap-4 flex-1 justify-center bg-slate-50">
          <p className="text-sm text-slate-600 text-center">Please enter your email to start chatting.</p>
          <input 
            type="email" 
            placeholder="Email address"
            className="border border-slate-300 rounded px-3 py-2 focus:outline-none focus:border-orange-500 focus:ring-1 focus:ring-orange-500"
            value={email}
            onChange={e => setEmail(e.target.value)}
            required
          />
          <button type="submit" className="bg-orange-600 text-white py-2 rounded font-medium hover:bg-orange-700 transition-colors">
            Start Chat
          </button>
        </form>
      ) : (
        <>
          <div className="flex-1 overflow-y-auto flex flex-col-reverse p-4 gap-3 bg-slate-50">
            {reversedMessages.map((m) => {
              const isSelf = m.sender_id === session.user_id;
              return (
                <div key={m.id} className={`flex ${isSelf ? 'justify-end' : 'justify-start'}`}>
                  <div className={`max-w-[85%] px-4 py-2 text-sm shadow-sm ${
                    isSelf 
                      ? 'bg-orange-600 text-white rounded-l-lg rounded-tr-lg' 
                      : 'bg-slate-100 text-slate-800 border border-slate-200 rounded-r-lg rounded-tl-lg'
                  }`}>
                    {m.content}
                  </div>
                </div>
              );
            })}
            {reversedMessages.length === 0 && <p className="text-center text-slate-400 text-sm mt-4 w-full">Send a message to start.</p>}
          </div>
          
          <form onSubmit={handleSend} className="p-3 bg-white border-t border-slate-100 flex items-center gap-2 shrink-0">
            <input 
              type="text" 
              placeholder="Type a message..."
              className="flex-1 bg-slate-100 rounded-full px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-orange-500"
              value={inputText}
              onChange={e => setInputText(e.target.value)}
            />
            <button type="submit" disabled={!isConnected || !inputText.trim()} className="bg-orange-600 text-white p-2 rounded-full hover:bg-orange-700 transition-colors disabled:opacity-50">
              <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 transform rotate-90" viewBox="0 0 20 20" fill="currentColor">
                <path d="M10.894 2.553a1 1 0 00-1.788 0l-7 14a1 1 0 001.169 1.409l5-1.429A1 1 0 009 15.571V11a1 1 0 112 0v4.571a1 1 0 00.725.962l5 1.428a1 1 0 001.17-1.408l-7-14z" />
              </svg>
            </button>
          </form>
        </>
      )}
    </div>
  );
};
