resource "azurerm_security_center_subscription_pricing" "servers" {
  tier          = "Free"
  resource_type = "VirtualMachines"

  extension {
    name = "AgentlessVmScanning"
  }
}
