@description('Name of the Web PubSub service')
param serviceName string = 'contoso-webpubsub-default'

@description('Deployment location')
param location string = resourceGroup().location

// publicNetworkAccess is omitted; the service defaults to Enabled.
resource webPubSub 'Microsoft.SignalRService/webPubSub@2023-08-01-preview' = {
  name: serviceName
  location: location
  sku: {
    name: 'Standard_S1'
    tier: 'Standard'
    capacity: 1
  }
  properties: {
    disableLocalAuth: true
  }
}
