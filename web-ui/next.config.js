/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  // Proxy API calls to the Go factory server (cmd/asf-web). Enabled in
  // development automatically; in production only when ASF_API_URL is set
  // explicitly, so a deployed dashboard never silently exposes a
  // localhost-only service.
  async rewrites() {
    const target = process.env.ASF_API_URL
      ?? (process.env.NODE_ENV === 'development' ? 'http://localhost:8080' : null)
    if (!target) return []
    return [
      {
        source: '/api/:path*',
        destination: `${target}/api/:path*`,
      },
    ]
  },
}

module.exports = nextConfig