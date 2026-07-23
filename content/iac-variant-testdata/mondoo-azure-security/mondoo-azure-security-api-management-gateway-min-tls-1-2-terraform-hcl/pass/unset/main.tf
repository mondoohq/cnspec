resource "azurerm_api_management" "example" {
  name                = "example-apim"
  location            = "eastus"
  resource_group_name = "example-rg"
  publisher_name      = "Example"
  publisher_email     = "admin@example.com"
  sku_name            = "Developer_1"

  security {
    enable_backend_ssl30 = false
    tls_ecdhe_ecdsa_with_aes_128_cbc_sha_ciphers_enabled = false
  }
}
