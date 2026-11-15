/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  // Proxy API calls to the Go factory server (cmd/asf-web) in development.
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${process.env.ASF_API_URL ?? 'http://localhost:8080'}/api/:path*`,
      },
    ]
  },
}

module.exports = nextConfig