import type { NextConfig } from "next";

const config: NextConfig = {
  agentRules: false,
  turbopack: { root: process.cwd() },
};

export default config;
