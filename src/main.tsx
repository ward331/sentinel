import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'

// Note: StrictMode removed — Leaflet MapContainer does not survive
// React 18 StrictMode's double mount/unmount cycle.
createRoot(document.getElementById('root')!).render(<App />)
