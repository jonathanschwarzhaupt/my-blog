# Merge public and admin binaries into one, gated by a feature flag

Status: supersedes ADR-0001

`blog` and `blog-admin` are merged into a single binary (`cmd/blog`; `cmd/blog-admin` is retired), with admin routes (compose/edit forms, project creation) gated behind a `-features` flag (comma-separated, e.g. `-features=admin`) rather than being absent from the compiled binary entirely. The same binary is deployed twice in the homelab: once with `-features=admin`, reachable only over Tailscale; once without it, exposed publicly via Cloudflare Tunnel.

This reverses ADR-0001's original security enforcement: two binaries meant admin routes literally didn't exist in the publicly-deployed process. A runtime flag means they're always compiled in and reachable in principle if the flag or a routing bug misbehaves — a real, accepted reduction in defense-in-depth, not an oversight. Chosen deliberately for a single-operator homelab context, in exchange for one codebase instead of two structurally near-identical ones, and because it aligns with a planned Helm chart (a `features:` values array rendered into this same flag) deployed via GitOps tooling (ArgoCD, managed by Flux) — a comma-separated feature list maps onto that directly, the way maintaining parallel binaries never would.
