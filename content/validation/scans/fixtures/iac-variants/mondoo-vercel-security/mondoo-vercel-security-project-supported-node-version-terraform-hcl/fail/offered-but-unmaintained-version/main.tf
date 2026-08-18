# Vercel still offers 20.x, but Node.js 20 left maintenance on 2026-04-30. A denylist of
# expired versions passed this; an allowlist of maintained ones fails it.
resource "vercel_project" "storefront" {
  name         = "storefront"
  node_version = "20.x"
}
