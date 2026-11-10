resource "azurerm_security_center_subscription_pricing" "ai" {
  tier          = "Standard"
  resource_type = "AI"

  extension {
    name = "AIPromptEvidence"
  }
}
