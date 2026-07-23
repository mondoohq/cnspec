resource nsgWriteAlert 'Microsoft.Insights/activityLogAlerts@2020-10-01' = {
  name: 'nsg-write-alert'
  location: 'global'
  properties: {
    enabled: true
    scopes: [
      '/subscriptions/00000000-0000-0000-0000-000000000000'
    ]
    condition: {
      allOf: [
        {
          field: 'category'
          equals: 'Administrative'
        }
        {
          field: 'operationName'
          equals: 'Microsoft.Network/networkSecurityGroups/write'
        }
      ]
    }
  }
}

resource nsgDeleteAlert 'Microsoft.Insights/activityLogAlerts@2020-10-01' = {
  name: 'nsg-delete-alert'
  location: 'global'
  properties: {
    enabled: true
    scopes: [
      '/subscriptions/00000000-0000-0000-0000-000000000000'
    ]
    condition: {
      allOf: [
        {
          field: 'category'
          equals: 'Administrative'
        }
        {
          field: 'operationName'
          equals: 'Microsoft.Network/networkSecurityGroups/delete'
        }
      ]
    }
  }
}
