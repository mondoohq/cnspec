resource topic 'Microsoft.EventGrid/topics@2023-12-15-preview' = {
  name: 'contoso-topic'
  location: 'eastus'
  properties: {
    inputSchema: 'EventGridSchema'
    publicNetworkAccess: 'Disabled'
    disableLocalAuth: true
  }
}
