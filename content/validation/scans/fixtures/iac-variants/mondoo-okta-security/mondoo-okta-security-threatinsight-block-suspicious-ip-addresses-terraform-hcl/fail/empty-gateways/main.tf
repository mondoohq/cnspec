provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_network_zone" "blocked_ips" {
  name     = "Blocked IP Addresses"
  type     = "IP"
  usage    = "BLOCKLIST"
  gateways = []
}
