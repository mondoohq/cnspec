@description('Name of the VPN connection')
param connectionName string = 's2s-conn-weakike'

@description('Deployment location')
param location string = resourceGroup().location

@description('Resource ID of the virtual network gateway')
param vnetGatewayId string

@description('Resource ID of the local network gateway')
param localGatewayId string

@description('IPsec shared key')
@secure()
param sharedKey string

// Uses a weak IKE encryption algorithm (AES128) for phase 1.
resource connection 'Microsoft.Network/connections@2023-09-01' = {
  name: connectionName
  location: location
  properties: {
    connectionType: 'IPsec'
    connectionProtocol: 'IKEv2'
    virtualNetworkGateway1: {
      id: vnetGatewayId
    }
    localNetworkGateway2: {
      id: localGatewayId
    }
    sharedKey: sharedKey
    usePolicyBasedTrafficSelectors: false
    ipsecPolicies: [
      {
        saLifeTimeSeconds: 27000
        saDataSizeKilobytes: 102400000
        ipsecEncryption: 'AES256'
        ipsecIntegrity: 'SHA256'
        ikeEncryption: 'AES128'
        ikeIntegrity: 'SHA256'
        dhGroup: 'DHGroup14'
        pfsGroup: 'PFS2'
      }
    ]
  }
}
