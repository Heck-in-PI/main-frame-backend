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
ScanAP path will use a wireless interface to scan for access 
```console
curl localhost:port/api/v1/modules/wifi/scanAp/INTERFACE_NAME
```

#### Deauth
:warning: __UNDERCONSTRUCTION__ :warning:\
Deauth path will deauthenticate user from an access point.\
Client mac can be specific mac address.
>NOTE: specific client mac is string in the format of a mac address e.g. FF:FF:FF:FF:FF:FF

```console
curl -X POST \
     -d "{\"apMac\" : \"AP_NAME\", \
          \"clientMac\" : \"FF:FF:FF:FF:FF:FF\", }" \
     localhost:port/api/v1/modules/wifi/deauth/INTERFACE_NAME
```
>This module in an extraction of bettercap deauth module and am not claiming the code

#### ConnectAP
ConnectAp path will connect you to an access point
```console
curl -X POST \
     -d "{\"apName\" : \"ssid\", \
          \"apPass\" : \"pass\"}" \
     localhost:port/api/v1/modules/wifi/connectAp/INTERFACE_NAME
```

#### CaptureHandshake
:warning: __UNDERCONSTRUCTION__ :warning:\
:warning: __NOT WORKING__ :warning:\
CaptureHandshake path will start capuring handshake all over the flore
This code is a mess in other turm *SPAGHETTI CODE*, I dont even know how I got it working but it is working KINDOF
>This module in an extraction of bettercap capture handshake module and am not claiming the code
