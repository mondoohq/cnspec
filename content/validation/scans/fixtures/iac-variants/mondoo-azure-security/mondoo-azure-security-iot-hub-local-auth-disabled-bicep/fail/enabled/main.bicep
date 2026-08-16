resource iotHub 'Microsoft.Devices/IotHubs@2023-06-30' = {
  name: 'iot-prod-001'
  location: 'eastus'
  sku: {
    name: 'S1'
    capacity: 1
  }
  properties: {
    disableLocalAuth: false
    minTlsVersion: '1.2'
  }
}
