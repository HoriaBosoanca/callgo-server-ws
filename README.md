# Callgo websocket API
## http
POST /initialize : nothing -> {"sessionID":"9Z83ZJHHR","password":"rZ8qZ1HHgz"} \
POST /disconnect : {"type": "leave", "memberID": "IdYUM1HHg", "memberName": "John"} -> disconnects a connected member's websocket
## websocket
/ws?sessionID=9Z83ZJHHR&displayName=John \
onInit -> to self {"type": "assignID", "memberID": "IdYUM1HHg", "memberName": "John"} && \
       -> to self foreach member in session except self {"type": "exist", "memberID": "[member's ID]", "memberName": "[member's name]"} && \
       -> to everyone except self {"type": "join", "memberID": "IdYUM1HHg", "memberName": "John"}
connected member sends {"to": "IdYUM1HHg", "from": "IdYUM1HHg", sdp: {"type": "offer/answer", "sdp": "[the sdp data]"}} -> to socket with id "to"
onDisconnect -> broadcast {"type": "leave", "memberID": "IdYUM1HHg", "memberName": "John"}
## links
[client](https://github.com/HoriaBosoanca/callgo-client)