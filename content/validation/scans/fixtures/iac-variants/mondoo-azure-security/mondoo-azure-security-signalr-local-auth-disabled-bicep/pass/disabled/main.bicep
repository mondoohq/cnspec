resource signalr 'Microsoft.SignalRService/signalR@2023-02-01' = {
  name: 'contoso-signalr'
  location: 'eastus'
  sku: {
    name: 'Standard_S1'
    tier: 'Standard'
    capacity: 1
  }
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    disableLocalAuth: true
    publicNetworkAccess: 'Disabled'
  }
}
