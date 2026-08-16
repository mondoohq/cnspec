resource functionApp 'Microsoft.Web/sites@2023-01-01' = {
  name: 'event-processor'
  location: 'eastus'
  kind: 'functionapp'
  properties: {
    siteConfig: {
      remoteDebuggingEnabled: false
      ftpsState: 'FtpsOnly'
    }
  }
}
