import type { NextConfig } from "next";

const backendUrl = process.env.BACKEND_URL || "http://localhost:8080";

const config: NextConfig = {
  agentRules: false,
  output: "standalone",
  turbopack: { root: process.cwd() },
  async rewrites() {
    return [
      {
        source: "/:service(carpool\\.v1\\.[^/]+)/:method",
        destination: `${backendUrl}/:service/:method`,
      },
    ];
  },
};

export default config;
