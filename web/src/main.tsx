import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { MantineProvider, createTheme } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import '@mantine/core/styles.css';
import '@mantine/notifications/styles.css';
import './index.css';
import App from './App.tsx';

const theme = createTheme({
  primaryColor: 'teal',
  fontFamily: '"IBM Plex Sans", "Segoe UI", sans-serif',
  defaultRadius: 'md',
  headings: {
    fontFamily: '"IBM Plex Sans", "Segoe UI", sans-serif',
    fontWeight: '600',
  },
  colors: {
    teal: [
      '#f1f8f5',
      '#dceee6',
      '#b5dcc9',
      '#8ac9ab',
      '#66b891',
      '#4aa87e',
      '#1f6b4a',
      '#1a5a3f',
      '#154a34',
      '#0f3a28',
    ],
  },
});

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <MantineProvider theme={theme} defaultColorScheme="light">
      <Notifications position="top-right" />
      <App />
    </MantineProvider>
  </StrictMode>,
);
