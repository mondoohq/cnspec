# No vercel_firewall_config is declared, so the firewall keeps its platform defaults, but a system
# bypass covering the whole internet still takes every request around it.
resource "vercel_project" "storefront" {
  name = "storefront"
}

resource "vercel_firewall_bypass" "everyone" {
  project_id = vercel_project.storefront.id
  domain     = "storefront.example.com"
  source_ip  = "0.0.0.0/0"
}
