@description('Name of the Data Factory')
param factoryName string = 'contoso-adf-default'

@description('Deployment location')
param location string = resourceGroup().location

resource dataFactory 'Microsoft.DataFactory/factories@2018-06-01' = {
  name: factoryName
  location: location
  identity: {
    type: 'SystemAssigned'
  }
  properties: {}
}
