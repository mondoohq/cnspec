# Free tier is only set for a non-storage plan; the storage assertion is vacuously
# satisfied because no StorageAccounts pricing plan opts out of Standard.
resource "azurerm_security_center_subscription_pricing" "vms" {
  tier          = "Free"
  resource_type = "VirtualMachines"
}
