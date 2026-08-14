resource ruleGroup 'Microsoft.Network/firewallPolicies/ruleCollectionGroups@2023-09-01' = {
  name: 'prod-rules'
  properties: {
    priority: 500
    ruleCollections: [
      {
        name: 'allow-app-to-db'
        ruleCollectionType: 'FirewallPolicyFilterRuleCollection'
        priority: 400
        action: {
          type: 'Allow'
        }
        rules: [
          {
            name: 'app-to-sql'
            ruleType: 'NetworkRule'
            ipProtocols: [
              'TCP'
            ]
            sourceAddresses: [
              '10.0.1.0/24'
            ]
            destinationAddresses: [
              '10.0.2.10'
            ]
            destinationPorts: [
              '1433'
            ]
          }
        ]
      }
    ]
  }
}
