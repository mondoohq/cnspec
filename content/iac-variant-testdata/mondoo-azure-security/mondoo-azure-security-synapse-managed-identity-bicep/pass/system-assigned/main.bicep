resource synapse 'Microsoft.Synapse/workspaces@2021-06-01' = {
  name: 'analytics-synapse'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    defaultDataLakeStorage: {
      accountUrl: 'https://exampledatalake.dfs.core.windows.net'
      filesystem: 'analytics'
    }
    trustedServiceBypassEnabled: false
  }
}
