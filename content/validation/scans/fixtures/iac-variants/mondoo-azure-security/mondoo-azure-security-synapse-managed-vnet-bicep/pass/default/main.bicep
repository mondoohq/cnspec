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
    azureADOnlyAuthentication: true
    managedVirtualNetwork: 'default'
    managedVirtualNetworkSettings: {
      preventDataExfiltration: true
    }
  }
}
