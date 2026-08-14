// A literal connection string in app settings is readable by anyone with
// Contributor on the app, and it lands in deployment history as well.
resource functionApp 'Microsoft.Web/sites@2023-01-01' = {
  name: 'event-processor'
  location: 'eastus'
  kind: 'functionapp'
  properties: {
    siteConfig: {
      appSettings: [
        {
          name: 'FUNCTIONS_WORKER_RUNTIME'
          value: 'python'
        }
        {
          name: 'DB_CONNECTIONSTRING'
          value: 'Server=tcp:example.database.windows.net;User ID=app;Password=P@ssw0rd-not-a-real-secret;'
        }
      ]
    }
  }
}
