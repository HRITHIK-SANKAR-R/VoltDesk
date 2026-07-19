import React, { useRef } from 'react';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { useGSAP } from '@gsap/react';

gsap.registerPlugin(ScrollTrigger);

export const LandingPage: React.FC = () => {
  const containerRef = useRef<HTMLDivElement>(null);
  const heroRef = useRef<HTMLDivElement>(null);
  const cardsRef = useRef<HTMLDivElement>(null);

  useGSAP(() => {
    // Hero Animation
    gsap.fromTo(
      heroRef.current,
      { opacity: 0, y: 50 },
      { opacity: 1, y: 0, duration: 1, ease: 'power3.out' }
    );

    // Feature Cards ScrollTrigger Animation
    const cards = gsap.utils.toArray('.feature-card');
    cards.forEach((card: any, i) => {
      gsap.fromTo(
        card,
        { opacity: 0, x: -50 },
        {
          opacity: 1,
          x: 0,
          duration: 0.8,
          ease: 'power3.out',
          scrollTrigger: {
            trigger: card,
            start: 'top 80%',
            toggleActions: 'play none none reverse',
          },
        }
      );
    });
  }, { scope: containerRef });

  return (
    <div ref={containerRef} className="min-h-screen bg-slate-50 text-slate-900 font-sans overflow-x-hidden selection:bg-orange-200">
      
      {/* Navbar */}
      <nav className="w-full px-8 py-6 flex justify-between items-center fixed top-0 bg-white/80 backdrop-blur-md z-50 border-b border-slate-200">
        <div className="text-2xl font-bold tracking-tight text-orange-600">VoltDesk<span className="text-slate-900">.</span></div>
        <div>
          <a href="http://localhost:8081/api/auth/google/login" className="bg-slate-900 hover:bg-slate-800 text-white px-5 py-2.5 rounded-full font-medium transition-colors text-sm shadow-sm inline-flex items-center gap-2">
            <svg viewBox="0 0 24 24" className="w-4 h-4 fill-current">
              <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
              <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
              <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
              <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
            </svg>
            Sign in with Google
          </a>
        </div>
      </nav>

      {/* Hero Section */}
      <section ref={heroRef} className="pt-40 pb-20 px-8 flex flex-col items-center justify-center text-center min-h-[80vh]">
        <div className="inline-block bg-orange-100 text-orange-700 px-4 py-1.5 rounded-full text-sm font-semibold mb-6 uppercase tracking-widest">
          The Future of Customer Support
        </div>
        <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-4xl text-slate-900 leading-tight">
          Resolve tickets <span className="text-transparent bg-clip-text bg-gradient-to-r from-orange-500 to-rose-500">faster</span> with intelligent real-time chat.
        </h1>
        <p className="mt-6 text-lg md:text-xl text-slate-500 max-w-2xl">
          VoltDesk provides an ultra-lean, WebSocket-driven support matrix powered by Google Gemini AI and Cloudflare R2 cold storage archiving.
        </p>
        <div className="mt-10">
          <a href="http://localhost:8081/api/auth/google/login" className="bg-orange-600 hover:bg-orange-700 text-white px-8 py-4 rounded-full font-bold transition-all hover:shadow-lg text-lg inline-flex items-center gap-2 hover:-translate-y-1">
            Start using VoltDesk free
            <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clipRule="evenodd" />
            </svg>
          </a>
        </div>
      </section>

      {/* Feature Cards Section */}
      <section ref={cardsRef} className="py-24 bg-slate-900 text-white">
        <div className="max-w-6xl mx-auto px-8">
          <h2 className="text-3xl md:text-5xl font-bold mb-16 text-center">Built for Speed and Scale</h2>
          
            <div className="grid md:grid-cols-3 gap-8">
            {/* Card 1 */}
            <div className="feature-card bg-slate-800 p-8 rounded-2xl border border-slate-700 shadow-xl">
              <div className="w-12 h-12 bg-blue-500/20 text-blue-400 flex items-center justify-center rounded-lg mb-6">
                <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <h3 className="text-xl font-bold mb-3">Real-Time WebSocket Sync</h3>
              <p className="text-slate-400">Powered by Go's high-throughput goroutines, messages are broadcast instantly with guaranteed transactional persistence.</p>
            </div>

            {/* Card 2 */}
            <div className="feature-card bg-slate-800 p-8 rounded-2xl border border-slate-700 shadow-xl">
              <div className="w-12 h-12 bg-purple-500/20 text-purple-400 flex items-center justify-center rounded-lg mb-6">
                <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
                </svg>
              </div>
              <h3 className="text-xl font-bold mb-3">Asynchronous Gzip Archiving</h3>
              <p className="text-slate-400">Background cron workers automatically compress and migrate 30-day old chats to Cloudflare R2 for zero-cost cold storage.</p>
            </div>

            {/* Card 3 */}
            <div className="feature-card bg-slate-800 p-8 rounded-2xl border border-slate-700 shadow-xl">
              <div className="w-12 h-12 bg-green-500/20 text-green-400 flex items-center justify-center rounded-lg mb-6">
                <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
                </svg>
              </div>
              <h3 className="text-xl font-bold mb-3">Zero-GC Memory Management</h3>
              <p className="text-slate-400">Engineered with extreme low-level optimizations. Features CPU-packed structs and sync.Pool byte buffers to virtually eliminate garbage collection latency.</p>
            </div>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-8 text-center text-slate-500 text-sm">
        &copy; {new Date().getFullYear()} VoltDesk. All rights reserved.
      </footer>
    </div>
  );
};
