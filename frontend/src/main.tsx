import React from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import './index.css'
import App from './ui/App'
import Home from './pages/Home'
import Showcase from './pages/Showcase'
import Docs from './pages/Docs'

const router = createBrowserRouter([
  { path: '/', element: <App />, children: [
    { index: true, element: <Home /> },
    { path: 'showcase', element: <Showcase /> },
    { path: 'docs', element: <Docs /> },
  ]}
])

createRoot(document.getElementById('root')!).render(<RouterProvider router={router} />)
