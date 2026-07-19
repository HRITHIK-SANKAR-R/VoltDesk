import { useState, useEffect } from 'react'
import './App.css'
import { CustomerWidget } from './components/CustomerWidget'
import { AgentDashboard } from './components/AgentDashboard'
import { LandingPage } from './components/LandingPage'

function App() {
  const [session, setSession] = useState<{ user_id: string; role: string; conversation_id?: string } | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // Check if user is authenticated via cookie by hitting /api/auth/me
    fetch('http://localhost:8081/api/auth/me')
      .then(res => {
        if (!res.ok) throw new Error('Not authenticated')
        return res.json()
      })
      .then(async (data) => {
        setSession(data)
      })
      .catch(() => {
        setSession(null)
      })
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return <div className="w-screen h-screen flex items-center justify-center text-slate-500">Loading...</div>
  }

  if (!session) {
    return <LandingPage />
  }

  if (session.role === 'customer') {
    return (
      <div className="w-full h-full flex items-center justify-center p-8 text-center text-slate-500 bg-slate-100 min-h-screen">
        <p className="max-w-md">
          You are authenticated as a customer. <br/><br/>
          Click the orange support icon in the bottom right corner to start a conversation.
        </p>
        <CustomerWidget userId={session.user_id} />
      </div>
    )
  }

  return <AgentDashboard />
}

export default App
