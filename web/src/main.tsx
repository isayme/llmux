import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App.tsx'
import './index.css'

const theme = localStorage.getItem('theme') || 'simple-light'
document.documentElement.setAttribute('data-theme', theme)

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter basename='/admin/'>
      <App />
    </BrowserRouter>
  </StrictMode>,
)