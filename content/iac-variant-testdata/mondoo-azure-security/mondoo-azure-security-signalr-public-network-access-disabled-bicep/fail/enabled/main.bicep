resource signalr 'Microsoft.SignalRService/signalR@2023-02-01' = {
  name: 'contoso-signalr'
  location: 'eastus'
  sku: {
    name: 'Standard_S1'
    tier: 'Standard'
    capacity: 1
  }
  properties: {
    publicNetworkAccess: 'Enabled'
  }
}
