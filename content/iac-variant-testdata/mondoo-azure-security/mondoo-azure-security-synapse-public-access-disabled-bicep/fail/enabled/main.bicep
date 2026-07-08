resource synapse 'Microsoft.Synapse/workspaces@2021-06-01' = {
  name: 'contosoanalytics'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    defaultDataLakeStorage: {
      accountUrl: 'https://contosodatalake.dfs.core.windows.net'
      filesystem: 'analytics'
    }
    sqlAdministratorLogin: 'sqladminuser'
    publicNetworkAccess: 'Enabled'
  }
}
