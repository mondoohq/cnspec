# No node_version pins the project, so it builds on whatever Vercel currently defaults to.
resource "vercel_project" "storefront" {
  name = "storefront"
}
