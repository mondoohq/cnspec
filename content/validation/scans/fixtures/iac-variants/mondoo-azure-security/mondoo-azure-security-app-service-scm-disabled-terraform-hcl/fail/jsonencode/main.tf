resource "azapi_update_resource" "scm_auth" {
  type        = "Microsoft.Web/sites/basicPublishingCredentialsPolicies@2022-09-01"
  resource_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Web/sites/example-web-app/basicPublishingCredentialsPolicies/scm"

  body = jsonencode({
    properties = {
      allow = true
    }
  })
}
