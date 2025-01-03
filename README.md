# Callgo websocket API
## http
POST /initialize : nothing -> {"sessionID":"9Z83ZJHHR","password":"rZ8qZ1HHgz"} \
POST /disconnect : {"sessionID":"9Z83ZJHHR","memberID":"ZUOeZJNNR","password":"rZ8qZ1HHgz"} -> disconnects a connected member's websocket
## websocket
/ws?sessionID=9Z83ZJHHR&displayName=John \
onInit -> {"myID":"IdYUM1HHg"} && broadcast foreach member in session {"InitID":"abcdef", "InitName":"John"} \
connected member sends {"video":"12345"} -> broadcast {"name":"John","memberID":"IdYUM1HHg","video":"12345"} \
onDisconnect -> broadcast {"disconnectID":"IdYUM1HHg"}
## links
[client](https://github.com/HoriaBosoanca/callgo-client)