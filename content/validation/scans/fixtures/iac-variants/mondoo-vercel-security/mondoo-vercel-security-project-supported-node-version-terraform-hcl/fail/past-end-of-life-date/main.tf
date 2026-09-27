# Node.js 20 is still offered by Vercel, but its upstream end-of-life date of 30 April 2026 has passed.
resource "vercel_project" "storefront" {
  name         = "storefront"
  node_version = "20.x"
}
