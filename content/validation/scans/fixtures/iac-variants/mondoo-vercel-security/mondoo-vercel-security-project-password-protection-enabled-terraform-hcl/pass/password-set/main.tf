variable "preview_password" {
  type      = string
  sensitive = true
}

resource "vercel_project" "storefront" {
  name = "storefront"

  password_protection = {
    deployment_type = "standard_protection_new"
    password        = var.preview_password
  }
}
