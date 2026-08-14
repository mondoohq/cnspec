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
    enablePurgeProtection: true
    softDeleteRetentionInDays: 90
    networkAcls: {
      bypass: 'AzureServices'
      defaultAction: 'Deny'
    }
  }
}
