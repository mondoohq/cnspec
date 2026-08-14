// Trusted service bypass lets any Azure service instance reach the workspace's
// storage regardless of the firewall rules configured on it.
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
    trustedServiceBypassEnabled: true
  }
}
