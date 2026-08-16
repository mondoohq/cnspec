resource hsm 'Microsoft.KeyVault/managedHSMs@2023-07-01' = {
  name: 'contosohsm'
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
    publicNetworkAccess: 'Enabled'
    enablePurgeProtection: true
    softDeleteRetentionInDays: 90
  }
}
