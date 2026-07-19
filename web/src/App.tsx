import { useState } from 'react'
import './App.css'
import { CustomerWidget } from './components/CustomerWidget'
import { AgentDashboard } from './components/AgentDashboard'

function App() {
  const [view, setView] = useState<'customer' | 'agent'>('customer')

  return (
    <div className="w-screen h-screen relative bg-slate-100 overflow-hidden">
      {/* Demo View Switcher */}
      <div className="absolute top-4 left-1/2 -translate-x-1/2 z-[100] bg-white rounded-full shadow-md p-1 flex gap-1 border border-slate-200">
        <button 
          onClick={() => setView('customer')}
          className={`px-4 py-1.5 rounded-full text-sm font-medium transition-colors ${view === 'customer' ? 'bg-orange-600 text-white' : 'text-slate-600 hover:bg-slate-50'}`}
        >
          Customer View
        </button>
        <button 
          onClick={() => setView('agent')}
          className={`px-4 py-1.5 rounded-full text-sm font-medium transition-colors ${view === 'agent' ? 'bg-slate-800 text-white' : 'text-slate-600 hover:bg-slate-50'}`}
        >
          Agent View
        </button>
      </div>

      {view === 'customer' ? (
        <div className="w-full h-full flex items-center justify-center p-8 text-center text-slate-500">
          <p className="max-w-md">
            You are viewing the host website as a customer. <br/><br/>
            Click the orange support icon in the bottom right corner to start a conversation.
          </p>
          <CustomerWidget />
        </div>
      ) : (
        <AgentDashboard />
      )}
    </div>
  )
}

export default App
