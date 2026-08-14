// A JIT policy covering no virtual machines leaves every management port open
// on its normal schedule.
resource jitPolicy 'Microsoft.Security/locations/jitNetworkAccessPolicies@2020-01-01' = {
  name: 'default'
  properties: {
    virtualMachines: []
  }
}
