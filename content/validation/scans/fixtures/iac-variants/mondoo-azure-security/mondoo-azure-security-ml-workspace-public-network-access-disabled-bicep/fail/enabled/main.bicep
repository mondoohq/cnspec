resource workspace 'Microsoft.MachineLearningServices/workspaces@2023-10-01' = {
  name: 'mlworkspace'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  sku: {
    name: 'Basic'
    tier: 'Basic'
  }
  properties: {
    friendlyName: 'ML Workspace'
    keyVault: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ml/providers/Microsoft.KeyVault/vaults/mlkv'
    storageAccount: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ml/providers/Microsoft.Storage/storageAccounts/mlstorage'
    applicationInsights: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ml/providers/Microsoft.Insights/components/mlai'
    publicNetworkAccess: 'Enabled'
  }
}
