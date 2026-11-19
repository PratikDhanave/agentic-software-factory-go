import './globals.css';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'asffactory — Agentic Software Factory Control Plane',
  description:
    'Autonomous Go Software Factory Control Plane with Explicit State Machines, Google Gemini Model Gateway, 10-Agent DAG Governance, and Isolated Sandbox Verification.',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        <link
          href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;600&display=swap"
          rel="stylesheet"
        />
        <script
          dangerouslySetInnerHTML={{
            __html: `
              try {
                var t = localStorage.getItem('asf-theme');
                if (t === 'dark') document.documentElement.classList.add('dark');
                else document.documentElement.classList.remove('dark');
              } catch (e) {}
            `,
          }}
        />
      </head>
      <body className="min-h-screen antialiased selection:bg-[#DCE9DF] selection:text-[#181816] dark:selection:bg-emerald-500/30 dark:selection:text-white">
        {children}
      </body>
    </html>
  );
}
