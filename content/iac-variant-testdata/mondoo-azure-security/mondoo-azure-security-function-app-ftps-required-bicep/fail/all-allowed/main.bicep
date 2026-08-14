// AllAllowed permits plaintext FTP, so deployment credentials and function
// code cross the network unencrypted.
resource functionApp 'Microsoft.Web/sites@2023-01-01' = {
  name: 'event-processor'
  location: 'eastus'
  kind: 'functionapp'
  properties: {
    siteConfig: {
      ftpsState: 'AllAllowed'
    }
  }
}
