# Trusted IPs is not in use on this project, so there is no enforcement mode to get wrong.
resource "vercel_project" "storefront" {
  name = "storefront"

  vercel_authentication = {
    deployment_type = "standard_protection_new"
  }
}
