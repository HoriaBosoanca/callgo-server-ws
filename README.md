# Callgo websocket API
## http
POST /initialize : nothing -> {"sessionID":"9Z83ZJHHR","password":"rZ8qZ1HHgz"} \
POST /disconnect : {"sessionID":"9Z83ZJHHR","memberID":"ZUOeZJNNR","password":"rZ8qZ1HHgz"} -> disconnects a connected member's websocket
## websocket
/ws?sessionID=9Z83ZJHHR \
onInit -> {"memberID":"IdYUM1HHg"} \
connected member sends {"name":"Horia","ID":"IdYUM1HHg","video":"12345"} -> everyone in session receives {"name":"Horia","ID":"IdYUM1HHg","video":"12345"}
## links
[client](https://github.com/HoriaBosoanca/callgo-client)