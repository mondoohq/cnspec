@description('Name of the Databricks workspace')
param workspaceName string = 'contoso-adb-open'

@description('Deployment location')
param location string = resourceGroup().location

resource databricks 'Microsoft.Databricks/workspaces@2024-05-01' = {
  name: workspaceName
  location: location
  sku: {
    name: 'premium'
  }
  properties: {
    managedResourceGroupId: subscriptionResourceId('Microsoft.Resources/resourceGroups', 'databricks-managed-rg')
    publicNetworkAccess: 'Enabled'
  }
}
