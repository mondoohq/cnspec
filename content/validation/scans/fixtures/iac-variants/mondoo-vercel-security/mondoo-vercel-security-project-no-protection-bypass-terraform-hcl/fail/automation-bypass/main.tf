# The bypass secret lets any request presenting it reach the deployments without satisfying the
# Vercel Authentication configured above.
resource "vercel_project" "storefront" {
  name = "storefront"

  vercel_authentication = {
    deployment_type = "standard_protection_new"
  }
}

resource "vercel_project_protection_bypass" "e2e" {
  project_id = vercel_project.storefront.id
}
