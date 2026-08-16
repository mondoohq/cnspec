resource flowLog 'Microsoft.Network/networkWatchers/flowLogs@2023-09-01' = {
  name: 'NetworkWatcher_eastus/fl-nsg-app-prod'
  location: 'eastus'
  properties: {
    targetResourceId: resourceId('Microsoft.Network/networkSecurityGroups', 'nsg-app-prod')
    storageId: resourceId('Microsoft.Storage/storageAccounts', 'stflowlogsprod')
    enabled: true
    retentionPolicy: {
      days: 90
      enabled: true
    }
  }
}
