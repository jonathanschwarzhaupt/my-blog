# Split public and admin binaries

Status: superseded by ADR-0003

The blog needs a compose/edit interface reachable only by Jonathan, while the public site must be internet-facing. Rather than gating admin routes with application-level auth inside one Go binary, the project is split into two binaries sharing one Neon Postgres database: `blog` (public, exposed via Cloudflare Tunnel) and `blog-admin` (private, reachable only over Tailscale). This trades a slightly more complex deployment for security enforced at the network layer instead of in application code.
