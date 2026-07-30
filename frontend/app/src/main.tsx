import React from 'react';
import ReactDOM from 'react-dom/client';
import { Toaster } from 'sonner';

import { App } from './App';
import './i18n';
import './styles/global.css';

// React starts here. The HTML file only has <div id="root" />.
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
    <Toaster
      position="top-center"
      richColors
      closeButton
      toastOptions={{
        duration: 3500,
        style: {
          borderRadius: 'var(--radius-lg)',
          border: '1px solid var(--border)',
          background: 'var(--card-solid)',
          color: 'var(--ink)',
          boxShadow: 'var(--elev-2)',
          fontFamily: 'inherit',
        },
      }}
    />
  </React.StrictMode>,
);
