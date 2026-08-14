resource mlWorkspace 'Microsoft.MachineLearningServices/workspaces@2024-04-01' = {
  name: 'research-ml'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    friendlyName: 'Research'
    encryption: {
      status: 'Enabled'
      keyVaultProperties: {
        keyVaultArmId: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/prod/providers/Microsoft.KeyVault/vaults/example-kv'
        keyIdentifier: 'https://example-kv.vault.azure.net/keys/ml-cmk/0123456789abcdef'
      }
    }
  }
}
