

const nextConfig = {
  reactStrictMode: true,
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://localhost:8081/api/:path*',
      },
      {
        source: '/events',
        destination: 'http://localhost:8081/events',
      },

    ];
  },
};
