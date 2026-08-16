// Remote debugging opens a powerful channel into the running app; it is meant
// to be temporary and is routinely left on after a debug session.
resource functionApp 'Microsoft.Web/sites@2023-01-01' = {
  name: 'event-processor'
  location: 'eastus'
  kind: 'functionapp'
  properties: {
    siteConfig: {
      remoteDebuggingEnabled: true
    }
  }
}
