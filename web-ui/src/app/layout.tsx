import './globals.css';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'ASF Workbench — Autonomous Software Factory IDE',
  description:
    'Autonomous Go Software Factory IDE with Spec Synthesis, 10-Agent DAG Governance, Isolated Workspace Verification, and SQLite Continuous Learning.',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
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
      <body className="min-h-screen antialiased selection:bg-blue-600/15 selection:text-blue-700 dark:selection:bg-blue-500/25 dark:selection:text-blue-200">
        {children}
      </body>
    </html>
  );
}
