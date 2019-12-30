let _leaveSession = null

async function initRpcClient() {
    let client = hprose.WebSocketClient.create(`ws://${document.location.host}/`, {Sessions: ["New", "Join", "Close", "Leave"]},
        {timeout: 86400000, idempotent: true, simple: true})
    let client_id = await client["#"]()
    return {client, client_id}
}

async function newSession() {
    const {client, client_id} = await initRpcClient()
    const {peerConnection, localStream, remoteStream} = await createPeerConnection(async desc => client.Sessions.New(client_id, desc.type, desc.sdp))
    _leaveSession = async () => {
        await client.Sessions.Close(client_id)
        peerConnection.close()
    }
    return {client_id, localStream, remoteStream}
}

async function joinSession(session_id) {
    const {client, client_id} = await initRpcClient()
    const {peerConnection, localStream, remoteStream} = await createPeerConnection(async desc => client.Sessions.Join(session_id, desc.type, desc.sdp))
    _leaveSession = async () => {
        await client.Sessions.Leave(session_id)
        peerConnection.close()
    }
    return {client_id, localStream, remoteStream}
}

async function leaveSession() {
    if (_leaveSession !== null) {
        await _leaveSession()
        _leaveSession = null
    }
}

async function createPeerConnection(getRemoteDescription) {
    const configuration = {iceServers: [{urls: "stun:stun.l.google.com:19302"}]} // RTCConfiguration
    const mediaConstraints = {video: {width: 640, height: 480, frameRate: 30}, audio: true} // MediaStreamConstraints
    let peerConnection = new RTCPeerConnection(configuration)

    let localStream = await navigator.mediaDevices.getUserMedia(mediaConstraints)
    localStream.getTracks().forEach(track => peerConnection.addTransceiver(track, {streams: [localStream], direction: "sendrecv"}))

    let offer = await peerConnection.createOffer()

    let desc = await getRemoteDescription(offer)
    peerConnection.onicecandidate = onICECandidate.bind(peerConnection, desc)
    let remoteStream = new MediaStream()
    peerConnection.ontrack = onTrack.bind(peerConnection, remoteStream)

    await peerConnection.setLocalDescription(offer)
    return {peerConnection, localStream, remoteStream}
}

async function onICECandidate(desc, event) {
    if (event.candidate === null) {
        await this.setRemoteDescription(desc)
    }
}

function onTrack(remoteStream, event) {
    event.streams.forEach(stream => stream.getTracks().forEach(track => remoteStream.addTrack(track)))
}
