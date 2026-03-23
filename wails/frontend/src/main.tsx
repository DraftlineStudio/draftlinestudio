import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './styles/global.css'
import { loadDictionary } from './services/spellCheck'

// Load spell check dictionary in background
loadDictionary()

// Disable default browser context menu everywhere (prevents WebView inspector)
// Custom context menus are handled by individual components
document.addEventListener('contextmenu', (e) => {
  const target = e.target as HTMLElement
  // Only allow for elements with data-context-menu (our custom menus)
  if (!target.closest('[data-context-menu]')) {
    e.preventDefault()
  }
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
