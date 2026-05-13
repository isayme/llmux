import { Routes, Route, Navigate } from 'react-router-dom'
import Login from './pages/Login'
import Layout from './components/Layout'
import Providers from './pages/Providers'
import ApiKeys from './pages/ApiKeys'
import Aliases from './pages/Aliases'

function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route element={<Layout />}>
        <Route path="/" element={<Navigate to="/providers" replace />} />
        <Route path="/providers" element={<Providers />} />
        <Route path="/api-keys" element={<ApiKeys />} />
        <Route path="/aliases" element={<Aliases />} />
      </Route>
    </Routes>
  )
}

export default App