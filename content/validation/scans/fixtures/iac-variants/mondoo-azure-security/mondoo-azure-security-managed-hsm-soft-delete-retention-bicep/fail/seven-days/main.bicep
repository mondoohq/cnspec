// A seven-day window gives very little time to notice and reverse a malicious
// or mistaken deletion of the HSM.
resource hsm 'Microsoft.KeyVault/managedHSMs@2023-07-01' = {
  name: 'prod-hsm'
  location: 'eastus'
  sku: {
    name: 'Standard_B1'
    family: 'B'
  }
  properties: {
    tenantId: '00000000-0000-0000-0000-000000000000'
    initialAdminObjectIds: [
      '11111111-1111-1111-1111-111111111111'
    ]
    softDeleteRetentionInDays: 7
  }
}
