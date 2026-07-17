# VoltDesk: UI/UX Design Specification

## 1. Design Philosophy
VoltDesk’s UI must reflect the engineering culture of high-growth, pragmatic companies like Sticker Mule. The interface must be:
*   **Information-Dense but Uncluttered:** Minimize padding where unnecessary; maximize readable text area.
*   **Keyboard-First:** Agents should be able to navigate the queue, accept AI drafts, and send messages without touching the mouse.
*   **Visually Distinct States:** AI-generated content must look fundamentally different from human-generated content to prevent accidental sends.

## 2. Design System & Tailwind Palette
We bypass heavy UI libraries (like Material UI or Ant Design) in favor of pure Tailwind CSS to demonstrate CSS mastery and keep the bundle size small.

*   **Typography:** System sans-serif (`font-sans`, default Tailwind stack). Clean and highly legible.
*   **Primary Action (Brand):** `bg-orange-600` (A subtle nod to Sticker Mule's brand color) for primary buttons and the customer chat widget bubble.
*   **Backgrounds:** `bg-slate-50` for main app backgrounds, `bg-white` for active panels.
*   **Text:** `text-slate-900` for primary text, `text-slate-500` for timestamps and secondary data.
*   **AI Elements:** `bg-purple-50` with `border-purple-200` and `text-purple-700` to clearly delineate AI-generated drafts from standard UI elements.

---

## 3. The Customer Widget (Frontend UI)
**Objective:** A frictionless, floating chat interface that does not block the host website's content.

### 3.1. Layout & Positioning
*   **Closed State:** A circular floating action button (FAB) fixed at `bottom-4 right-4` or `bottom-8 right-8`. Uses a chat bubble SVG icon.
    *   *Tailwind:* `fixed bottom-6 right-6 w-14 h-14 rounded-full bg-orange-600 shadow-lg hover:scale-105 transition-transform`
*   **Open State:** A fixed rectangular panel, growing from the bottom right.
    *   *Dimensions:* `w-80 h-96` (Max height restricted to `calc(100vh - 2rem)` for mobile screens).
    *   *Tailwind:* `fixed bottom-24 right-6 w-80 h-96 bg-white rounded-xl shadow-2xl flex flex-col overflow-hidden`

### 3.2. Widget Components
1.  **Header:** Solid `bg-orange-600` with white text. Displays "Support" and a minimize `-` button.
2.  **Message Thread (Scrollable Area):**
    *   *Container:* `flex-1 overflow-y-auto p-4 space-y-4 bg-slate-50`. Must implement a scroll-to-bottom anchor on new messages.
    *   *Customer Bubble (Self):* Sent by the user. Aligned right. `bg-orange-600 text-white rounded-l-lg rounded-tr-lg`.
    *   *Agent Bubble (Other):* Received from the agent. Aligned left. `bg-white border border-slate-200 text-slate-800 rounded-r-lg rounded-tl-lg`.
3.  **Input Area:** Fixed at the bottom of the widget.
    *   *Container:* `p-3 bg-white border-t border-slate-100 flex items-center gap-2`.
    *   *Input:* `flex-1 bg-slate-100 rounded-full px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-orange-500`.

---

## 4. The Agent Dashboard (Frontend UI)
**Objective:** A command center for support agents. It must support rapid context switching between conversations.

### 4.1. Layout Grid
A full-width, full-height SPA (`w-screen h-screen overflow-hidden flex`).

*   **Left Sidebar (The Queue):** `w-80 flex-shrink-0 bg-white border-r border-slate-200 flex flex-col`.
*   **Main Conversation Window:** `flex-1 bg-slate-50 flex flex-col relative`.

### 4.2. Sidebar (Chat Queue)
*   **Header:** Displays agent status and a filter toggle (Open/Resolved).
*   **Queue Items:** A vertical list of active conversations.
    *   *Active State:* `bg-orange-50 border-l-4 border-orange-600`.
    *   *Inactive State:* `bg-white hover:bg-slate-50`.
    *   *Data displayed:* Customer ID (truncated or pseudo-name), timestamp of the last message, and a 1-line truncation of the last message (`truncate text-sm text-slate-500`).
    *   *Notification Badge:* A red dot if the conversation has unread messages.

### 4.3. Main Conversation Window
1.  **Top Bar:** Displays the Customer ID, connection status (e.g., a green dot for "Online" via WebSocket), and a "Mark Resolved" button.
2.  **Message Thread:** Similar styling to the customer widget, but scaled up.
    *   *Agent Bubble (Self):* Aligned right, `bg-slate-800 text-white`.
    *   *Customer Bubble (Other):* Aligned left, `bg-white border border-slate-200 shadow-sm`.
3.  **The Input & AI Composer Area (Crucial UI/UX):**
    This sits at the bottom of the main window. It is split into two visual tiers.

---

## 5. The "Smart Draft" AI UI (Core Differentiator)
This is where the AI feature lives. When the Go backend sends an `is_ai_draft: true` payload, it does *not* render in the main chat thread. Instead, it populates a dedicated staging area above the text input.

### 5.1. Visual Specification
*   **Container:** Sits directly above the text input, separated by a slight margin.
    *   *Tailwind:* `mx-4 mb-2 p-4 bg-purple-50 border border-purple-200 border-dashed rounded-lg flex flex-col gap-2 relative`.
*   **Indicator:** A small sparkle icon ✨ and text `AI Suggested Reply` in `text-purple-600 text-xs font-semibold uppercase tracking-wider`.
*   **Draft Content:** The actual Gemini-generated text. Editable. `text-slate-800 text-sm`.
*   **Action Row:**
    *   **Send (Accept):** `bg-purple-600 hover:bg-purple-700 text-white px-3 py-1.5 rounded-md text-sm font-medium transition-colors`.
    *   **Discard:** `text-slate-500 hover:text-slate-700 text-sm font-medium`.

### 5.2. UX Interaction Flow
1.  Customer sends: "Where is my order?"
2.  Agent sees the message appear in the thread.
3.  *Wait 1-2 seconds.*
4.  The "Smart Draft" box slides up (`animate-slide-up`) above the input field, pre-populated with: "I'd be happy to check on that for you. Do you have your order number?"
5.  **Keyboard Shortcut:** The agent can press `Tab` to focus the draft, edit it if necessary, and press `Cmd+Enter` (or `Ctrl+Enter`) to send it immediately.
6.  Once sent, the draft box collapses, and the text becomes a standard Agent Bubble in the main thread.

## 6. Micro-Interactions & States
*   **WebSocket Connecting:** When the React app boots, show a subtle loading spinner in the header.
*   **WebSocket Disconnected:** A red banner drops down from the top: `Lost connection to server. Reconnecting...`
*   **Typing Indicators:** When the customer is typing, display a small `typing...` animation (three bouncing dots) at the bottom of the agent's message thread.
*   **Focus Management:** When clicking on a conversation in the left sidebar, the browser focus must immediately snap to the chat input field, allowing the agent to type immediately without a second click.
