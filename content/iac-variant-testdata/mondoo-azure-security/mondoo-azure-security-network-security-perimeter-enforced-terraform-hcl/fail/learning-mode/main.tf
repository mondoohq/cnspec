# Learning mode observes and logs what the perimeter would block without
# blocking anything, so the resource remains reachable as before.
resource "azurerm_network_security_perimeter_association" "storage" {
  name                     = "storage-association"
  perimeter_id             = azurerm_network_security_perimeter.prod.id
  profile_id               = azurerm_network_security_perimeter_profile.prod.id
  private_link_resource_id = azurerm_storage_account.data.id
  access_mode              = "Learning"
}
