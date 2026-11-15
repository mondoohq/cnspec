targetScope = 'subscription'

resource serversPricing 'Microsoft.Security/pricings@2024-01-01' = {
  name: 'VirtualMachines'
  properties: {
    pricingTier: 'Standard'
    subPlan: 'P2'
    extensions: [
      {
        name: 'AgentlessVmScanning'
        isEnabled: 'False'
      }
    ]
  }
}
