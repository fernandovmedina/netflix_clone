import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "occ-0-7553-114.1.nflxso.net",
        pathname: "/dnm/api/v6/**",
      },
    ],
  },
};

export default nextConfig;
