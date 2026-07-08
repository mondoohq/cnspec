resource signalr 'Microsoft.SignalRService/signalR@2023-02-01' = {
  name: 'contoso-signalr'
  location: 'eastus'
  sku: {
    name: 'Free_F1'
    tier: 'Free'
    capacity: 1
  }
  properties: {
    disableLocalAuth: true
  }
}
