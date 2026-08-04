import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import '@fontsource/ibm-plex-sans/400.css'
import '@fontsource/ibm-plex-sans/500.css'
import '@fontsource/ibm-plex-sans/600.css'
import '@fontsource/merriweather/300.css'
import '@fontsource/merriweather/400.css'
import '@fontsource/merriweather/700.css'
import '@fontsource/merriweather/300-italic.css'
import '@fontsource/merriweather/400-italic.css'
import '@fontsource/merriweather/700-italic.css'
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
