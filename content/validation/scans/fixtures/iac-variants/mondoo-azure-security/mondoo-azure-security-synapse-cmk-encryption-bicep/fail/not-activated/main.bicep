resource synapseWorkspace 'Microsoft.Synapse/workspaces@2021-06-01' = {
  name: 'syn-contoso-prod'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    defaultDataLakeStorage: {
      accountUrl: 'https://stcontosodls.dfs.core.windows.net'
      filesystem: 'synapse'
    }
    managedVirtualNetwork: 'default'
    azureADOnlyAuthentication: true
    encryption: {
      cmk: {
        kekIdentity: {
          useSystemAssignedIdentity: true
        }
        key: {
          name: 'default'
          keyVaultUrl: 'https://kv-contoso-prod.vault.azure.net/keys/synapse-cmk'
        }
      }
    }
  }
}
