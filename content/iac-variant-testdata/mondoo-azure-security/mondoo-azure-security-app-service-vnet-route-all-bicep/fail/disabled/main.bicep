resource site 'Microsoft.Web/sites@2023-12-01' = {
  name: 'example-webapp'
  location: 'eastus'
  properties: {
    httpsOnly: true
  }
}

resource webConfig 'Microsoft.Web/sites/config@2023-12-01' = {
  parent: site
  name: 'web'
  properties: {
    vnetRouteAllEnabled: false
    scmMinTlsVersion: '1.2'
  }
}
