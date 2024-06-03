# main-frame backend
The backend that holds the raspberry pi HACKING modules

## Modules

### Wifi
:warning: __UNDERCONSTRUCTION__ :warning:\
Wifi module will be responsible of every thing related to wifi application 

#### Interfaces
Interfaces path will list device wirless interfaces, think of it as iwconfig.
```bash
curl localhost:port/api/v1/modules/wifi/interface
```

#### ScanAP
ScanAP path will put wireless interface in monitor mode and capture packets and filter for access point 
```bash
curl localhost:port/api/v1/modules/wifi/scanAp/INTERFACE_NAME
```
>This path start a goroutine and keep running in background

#### ScanClient
ScanClient path will use the wireless interface in monitor mode and capture packets and filter for the already found access point's client 
```bash
curl localhost:port/api/v1/modules/wifi/scanClient
```
>This path start a goroutine and keep running in background
>For this to work the ScanAP should be already running

#### Deauth
Deauth path will deauthenticate user from an access point.\
Client mac can be specific mac address.
>NOTE: specific client mac is string in the format of a mac address e.g. FF:FF:FF:FF:FF:FF

```bash
curl -X POST \
     -d "{\"apMac\" : \"FF:FF:FF:FF:FF:FF\", \
          \"clientMac\" : \"FF:FF:FF:FF:FF:FF\" }" \
     localhost:port/api/v1/modules/wifi/deauth
```

#### ConnectAP
ConnectAp path will connect you to an access point
```bash
curl -X POST \
     -d "{\"apName\" : \"ssid\", \
          \"apPass\" : \"pass\"}" \
     localhost:port/api/v1/modules/wifi/connectAp/INTERFACE_NAME
```

#### CaptureHandshake
CaptureHandshake path will use the wireless interface in monitor mode and capture packets and filter handshakes all over the flore
```bash
curl localhost:port/api/v1/modules/wifi/cptHandshake
```
>This path start a goroutine and keep running in background
>For this to work the ScanAP and ScanClient should be already running

#### Probe
Probe path will send a fake client probe with the given station BSSID, searching for ESSID.\
>NOTE: specific ap mac is string in the format of a mac address e.g. FF:FF:FF:FF:FF:FF

```bash
curl -X POST \
     -d "{\"apMac\" : \"FF:FF:FF:FF:FF:FF\", \
          \"apName\" : \"AP_NAME\" }" \
     localhost:port/api/v1/modules/wifi/probe
```

#### Beacon
Beacon path will send a fake fake management beacons in order to create N access point.

```bash
curl -X POST \
     -d "{\"numberOfAP\" : 3,\
          \"apName\" : \"AP_NAME\",\
          \"ApChannel\" : 1, \
          \"ApEncryption\" : true }" \
     localhost:port/api/v1/modules/wifi/beacon
```

#### RogueAP
RogueAP path will send a fake fake management beacons in order to create rogue access point.\
>NOTE: specific ap mac is string in the format of a mac address e.g. FF:FF:FF:FF:FF:FF

```bash
curl -X POST \
     -d "{\"apName\" : \"AP_NAME\",\
          \"apMac\" : \"FF:FF:FF:FF:FF:FF\", \
          \"ApChannel\" : 1, \
          \"ApEncryption\" : true }" \
     localhost:port/api/v1/modules/wifi/beacon
```

#### Stop
Stop path will kill all process of recon
```bash
curl localhost:port/api/v1/modules/wifi/stop
```

#### StopScanClient
StopScanClient path will kill process of searching for access point clients
```bash
curl localhost:port/api/v1/modules/wifi/stopScanClient
```

#### StopCptHandshake
StopCptHandshake path will kill process of searching access points handshakes 
```bash
curl localhost:port/api/v1/modules/wifi/stopCptHandshake
```

#### StopBeaconer
StopBeaconer path will kill process of sending beacons.
```bash
curl localhost:port/api/v1/modules/wifi/stopBeaconer
```

>ScanAp, ScanClient, Deauth, CaptureHandshake are either extracted from bettercap or inspired by them am not claiming the code please support the official release 
