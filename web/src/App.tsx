import { useEffect } from 'react'
import './App.css'
import { CustomerWidget } from './components/CustomerWidget'
import { AgentDashboard } from './components/AgentDashboard'
import { LandingPage } from './components/LandingPage'
import { AuthProvider, useAuth } from './context/AuthContext'

function AppContent() {
  const { user, loading } = useAuth()

  useEffect(() => {
    if (loading) {
      document.title = "VoltDesk | Loading...";
    } else if (!user) {
      document.title = "VoltDesk | Real-Time AI Support";
    } else if (user.role === 'customer') {
      document.title = `VoltDesk | Support (${user.name || user.email})`;
    } else if (user.role === 'agent') {
      document.title = `VoltDesk | Agent Console`;
    }
  }, [user, loading]);

  if (loading) {
    return <div className="w-screen h-screen flex items-center justify-center text-slate-400 bg-slate-900">Loading...</div>
  }

  if (!user) {
    return <LandingPage />
  }

  if (user.role === 'customer') {
    return (
      <>
        <LandingPage />
        <CustomerWidget userId={user.user_id} conversationId={user.conversation_id!} />
      </>
    )
  }

  return <AgentDashboard />
}

function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  )
}

export default App
