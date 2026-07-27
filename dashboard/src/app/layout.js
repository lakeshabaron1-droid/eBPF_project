import './globals.css';
import { Inter, JetBrains_Mono } from 'next/font/google';


const inter = Inter({
  subsets: ['latin'],
  variable: '--font-sans',
});


const jetbrainsMono = JetBrains_Mono({

  subsets: ['latin'],
  variable: '--font-mono',

});

export const metadata = {
  title: 'eBPF Zero-Trust Gateway Dashboard',
  description: 'Real-time telemetry & security controls',
};

export default function RootLayout({ children }) {
  return (
    <html lang="en" className={`${inter.variable} ${jetbrainsMono.variable}`}>
      <body>
        <div className="app-container">
          {children}
        </div>

