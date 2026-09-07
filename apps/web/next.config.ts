import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  experimental: {
    useTypeScriptCli: false,
  },
  async rewrites() {
    return [
      {
        source: "/api/v1/:path*",
        destination: `${process.env.API_URL ?? "http://127.0.0.1:8080"}/api/v1/:path*`,
      },
    ];
  },
};

export default nextConfig;
