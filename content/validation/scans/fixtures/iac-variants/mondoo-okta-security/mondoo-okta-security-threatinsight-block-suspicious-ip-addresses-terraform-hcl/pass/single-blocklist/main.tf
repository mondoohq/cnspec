provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_network_zone" "blocked_ips" {
  name     = "Blocked IP Addresses"
  type     = "IP"
  usage    = "BLOCKLIST"
  gateways = ["1.2.3.4/24", "2.3.4.5"]
}

resource "okta_network_zone" "trusted" {
  name     = "Corporate Network"
  type     = "IP"
  usage    = "POLICY"
  gateways = ["10.0.0.0/8"]
}
