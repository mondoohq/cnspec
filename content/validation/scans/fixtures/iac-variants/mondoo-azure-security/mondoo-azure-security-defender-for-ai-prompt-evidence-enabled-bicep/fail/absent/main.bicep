targetScope = 'subscription'

resource aiPricing 'Microsoft.Security/pricings@2024-01-01' = {
  name: 'AI'
  properties: {
    pricingTier: 'Standard'
  }
}
