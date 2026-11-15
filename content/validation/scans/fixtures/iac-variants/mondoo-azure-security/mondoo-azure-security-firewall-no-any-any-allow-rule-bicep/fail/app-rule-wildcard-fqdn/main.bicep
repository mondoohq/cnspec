// Any source may reach any FQDN over HTTPS.
resource ruleGroup 'Microsoft.Network/firewallPolicies/ruleCollectionGroups@2023-09-01' = {
  name: 'prod-rules'
  properties: {
    priority: 500
    ruleCollections: [
      {
        name: 'allow-all'
        ruleCollectionType: 'FirewallPolicyFilterRuleCollection'
        priority: 400
        action: {
          type: 'Allow'
        }
        rules: [
          {
            name: 'any-web'
            ruleType: 'ApplicationRule'
            protocols: [
              {
                protocolType: 'Https'
                port: 443
              }
            ]
            sourceAddresses: [
              '*'
            ]
            targetFqdns: [
              '*'
            ]
          }
        ]
      }
    ]
  }
}
