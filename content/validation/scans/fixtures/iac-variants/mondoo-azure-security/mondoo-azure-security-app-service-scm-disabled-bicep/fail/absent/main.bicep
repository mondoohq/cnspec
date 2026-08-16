resource site 'Microsoft.Web/sites@2023-12-01' = {
  name: 'example-webapp'
  location: 'eastus'
  properties: {
    httpsOnly: true
  }
}

resource scmPolicy 'Microsoft.Web/sites/basicPublishingCredentialsPolicies@2023-12-01' = {
  parent: site
  name: 'scm'
  properties: {
    updatable: true
  }
}
