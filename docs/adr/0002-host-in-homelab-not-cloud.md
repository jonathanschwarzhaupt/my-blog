# Host in the homelab, not the cloud

Both binaries run in Jonathan's homelab rather than a cloud VPS, coupling the blog's uptime to home network and hardware reliability. This is accepted deliberately: the homelab itself is a Project the blog writes about, so self-hosting is part of the portfolio's proof-of-work, not an accidental constraint. `blog` is exposed via Cloudflare Tunnel; `blog-admin` via Tailscale only.
