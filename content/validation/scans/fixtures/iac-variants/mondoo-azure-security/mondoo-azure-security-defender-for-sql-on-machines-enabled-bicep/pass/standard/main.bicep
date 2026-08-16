resource sqlVmPricing 'Microsoft.Security/pricings@2023-01-01' = {
  name: 'SqlServerVirtualMachines'
  properties: {
    pricingTier: 'Standard'
  }
}
