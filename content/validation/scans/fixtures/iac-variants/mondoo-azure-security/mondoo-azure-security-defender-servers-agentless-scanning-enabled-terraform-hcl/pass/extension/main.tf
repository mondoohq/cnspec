resource "azurerm_security_center_subscription_pricing" "servers" {
  tier          = "Standard"
  resource_type = "VirtualMachines"
  subplan       = "P2"

  extension {
    name = "AgentlessVmScanning"
  }
}
