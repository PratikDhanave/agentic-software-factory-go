/** @type {import('next').NextConfig} */
const isProd = process.env.NODE_ENV === 'production';

const nextConfig = isProd
  ? {
      reactStrictMode: true,
      output: 'export',
      images: { unoptimized: true },
    }
  : {
      reactStrictMode: true,
      async rewrites() {
        const target = process.env.ASF_API_URL || 'http://localhost:8080';
        return [
          {
            source: '/api/:path*',
            destination: `${target}/api/:path*`,
          },
          {
            source: '/webhooks/:path*',
            destination: `${target}/webhooks/:path*`,
          },
          {
            source: '/health',
            destination: `${target}/health`,
          },
        ];
      },
    };

module.exports = nextConfig;
