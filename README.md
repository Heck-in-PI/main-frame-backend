# main-frame backend
The backend that holds the raspberry pi HACKING modules

## Modules

### Wifi
:warning: __UNDERCONSTRUCTION__ :warning:\
Wifi module will be responsible of every thing related to wifi application 

#### Interfaces
Interfaces path will list device wirless interfaces, think of it as iwconfig.
```console
curl localhost:port/api/v1/modules/wifi/interface
```

#### ScanAP
ScanAP path will put wireless interface in monitor mode and capture packets and filter for access point 
```console
curl localhost:port/api/v1/modules/wifi/scanAp/INTERFACE_NAME
```
>This path start a goroutine and keep running in background

#### ScanClient
ScanClient path will use the wireless interface in monitor mode and capture packets and filter for the already found access point's client 
```console
curl localhost:port/api/v1/modules/wifi/scanClient
```
>This path start a goroutine and keep running in background
>For this to work the ScanAP should be already running

#### Deauth
Deauth path will deauthenticate user from an access point.\
Client mac can be specific mac address.
>NOTE: specific client mac is string in the format of a mac address e.g. FF:FF:FF:FF:FF:FF

```console
curl -X POST \
     -d "{\"apMac\" : \"AP_NAME\", \
          \"clientMac\" : \"FF:FF:FF:FF:FF:FF\", }" \
     localhost:port/api/v1/modules/wifi/deauth/INTERFACE_NAME
```

#### ConnectAP
ConnectAp path will connect you to an access point
```console
curl -X POST \
     -d "{\"apName\" : \"ssid\", \
          \"apPass\" : \"pass\"}" \
     localhost:port/api/v1/modules/wifi/connectAp/INTERFACE_NAME
```

#### CaptureHandshake
CaptureHandshake path will use the wireless interface in monitor mode and capture packets and filter handshakes all over the flore
```console
curl localhost:port/api/v1/modules/wifi/cptHandshake
```
>This path start a goroutine and keep running in background
>For this to work the ScanAP and ScanClient should be already running

#### Stop
Stop path will kill all process of recon
```console
curl localhost:port/api/v1/modules/wifi/stop
```

>ScanAp, ScanClient, Deauth, CaptureHandshake are either extracted from bettercap or inspired by them am not claiming the code please support the official release 
