resource wsA 'Microsoft.Synapse/workspaces@2021-06-01' = {
  name: 'syn-a'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    defaultDataLakeStorage: {
      accountUrl: 'https://stcontosodls.dfs.core.windows.net'
      filesystem: 'synapse'
    }
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

resource keyA 'Microsoft.Synapse/workspaces/keys@2021-06-01' = {
  name: 'syn-a/default'
  properties: {
    isActiveCMK: true
    keyVaultUrl: 'https://kv-contoso-prod.vault.azure.net/keys/synapse-cmk'
  }
}
