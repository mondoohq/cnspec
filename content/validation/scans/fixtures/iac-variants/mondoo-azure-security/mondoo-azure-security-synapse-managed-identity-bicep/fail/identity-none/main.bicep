// Without a managed identity the workspace authenticates to linked data stores
// with keys or SAS tokens held in its configuration.
resource synapse 'Microsoft.Synapse/workspaces@2021-06-01' = {
  name: 'analytics-synapse'
  location: 'eastus'
  identity: {
    type: 'None'
  }
  properties: {
    defaultDataLakeStorage: {
      accountUrl: 'https://exampledatalake.dfs.core.windows.net'
      filesystem: 'analytics'
    }
  }
}
