@description('Name of the Web PubSub service')
param serviceName string = 'contoso-webpubsub-open'

@description('Deployment location')
param location string = resourceGroup().location

resource webPubSub 'Microsoft.SignalRService/webPubSub@2023-08-01-preview' = {
  name: serviceName
  location: location
  sku: {
    name: 'Standard_S1'
    tier: 'Standard'
    capacity: 1
  }
  properties: {
    publicNetworkAccess: 'Enabled'
    disableLocalAuth: false
  }
}
