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
      <div className="ticks"></div>

      <section id="next-steps">
        <div id="docs">
          <svg className="icon" role="presentation" aria-hidden="true">
            <use href="/icons.svg#documentation-icon"></use>
          </svg>
          <h2>Documentation</h2>
          <p>Your questions, answered</p>
          <ul>
            <li>
              <a href="https://vite.dev/" target="_blank">
                <img className="logo" src={viteLogo} alt="" />
                Explore Vite
              </a>
            </li>
            <li>
              <a href="https://react.dev/" target="_blank">
                <img className="button-icon" src={reactLogo} alt="" />
                Learn more
              </a>
            </li>
          </ul>
        </div>
        <div id="social">
          <svg className="icon" role="presentation" aria-hidden="true">
            <use href="/icons.svg#social-icon"></use>
          </svg>
          <h2>Connect with us</h2>
          <p>Join the Vite community</p>
          <ul>
            <li>
              <a href="https://github.com/vitejs/vite" target="_blank">
                <svg
                  className="button-icon"
                  role="presentation"
                  aria-hidden="true"
                >
                  <use href="/icons.svg#github-icon"></use>
                </svg>
                GitHub
              </a>
            </li>
            <li>
              <a href="https://chat.vite.dev/" target="_blank">
                <svg
                  className="button-icon"
                  role="presentation"
                  aria-hidden="true"
                >
                  <use href="/icons.svg#discord-icon"></use>
                </svg>
                Discord
              </a>
            </li>
            <li>
              <a href="https://x.com/vite_js" target="_blank">
                <svg
                  className="button-icon"
                  role="presentation"
                  aria-hidden="true"
                >
                  <use href="/icons.svg#x-icon"></use>
                </svg>
                X.com
              </a>
            </li>
            <li>
              <a href="https://bsky.app/profile/vite.dev" target="_blank">
                <svg
                  className="button-icon"
                  role="presentation"
                  aria-hidden="true"
                >
                  <use href="/icons.svg#bluesky-icon"></use>
                </svg>
                Bluesky
              </a>
            </li>
          </ul>
        </div>
      </section>

      <div className="ticks"></div>
      <section id="spacer"></section>
    </>
  )
}

export default App
