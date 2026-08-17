# A project that declares system bypasses without a vercel_firewall_config is still in scope, and
# this one names a single monitoring host rather than the whole internet.
resource "vercel_project" "storefront" {
  name = "storefront"
}

resource "vercel_firewall_bypass" "monitoring" {
  project_id = vercel_project.storefront.id
  domain     = "storefront.example.com"
  source_ip  = "198.51.100.14"
}
