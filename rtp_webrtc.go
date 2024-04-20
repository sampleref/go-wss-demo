package main

import (
	"errors"
	"fmt"
	"github.com/pion/webrtc/v3"
	"io"
	"log"
	"net"
)

var clientPeerConnections map[string]*webrtc.PeerConnection

func ReadRTPAndGenerateSDPOffer(clientId string, rtpPort int) string {

	m := &webrtc.MediaEngine{}

	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000,
			SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42C01E"},
		PayloadType: 96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		panic(err)
	}
	clientPeerConnections[clientId] = peerConnection

	fmt.Printf("RTP 2 WebRTC attempt to listen udp/rtp for Client: %s\n", clientId)
	// Open a UDP Listener for RTP Packets on port
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: rtpPort})
	if err != nil {
		panic(err)
	}
	fmt.Printf("RTP 2 WebRTC updlistener created for Client: %s\n", clientId)

	// Increase the UDP receive buffer size
	// Default UDP buffer sizes vary on different operating systems
	// bufferSize := 300000 // 300KB
	/*err = listener.SetReadBuffer(bufferSize)
	if err != nil {
		panic(err)
	}*/

	// Create a video track
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "video1")
	if err != nil {
		panic(err)
	}
	rtpSender, err := peerConnection.AddTrack(videoTrack)

	if err != nil {
		panic(err)
	}

	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things
	// like NACK this needs to be called.

	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())

		if connectionState == webrtc.ICEConnectionStateFailed {
			if closeErr := peerConnection.Close(); closeErr != nil {
				panic(closeErr)
			}
		}
	})

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	// Create an offer to send to the other process
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	// Note: this will start the gathering of ICE candidates
	if err = peerConnection.SetLocalDescription(offer); err != nil {
		panic(err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete
	fmt.Printf("RTP 2 WebRTC gatherComplete for Client: %s\n", clientId)

	fmt.Printf("RTP 2 WebRTC Offer for Client: %s --- \n %s \n --- \n", clientId, offer.SDP)
	go loopRtpToVideoTrack(listener, videoTrack, clientId)
	return offer.SDP
}

func loopRtpToVideoTrack(listener *net.UDPConn, videoTrack *webrtc.TrackLocalStaticRTP, clientId string) {

	defer func() {
		if err := listener.Close(); err != nil {
			panic(err)
		}
	}()

	// Read RTP packets forever and send them to the WebRTC Client
	inboundRTPPacket := make([]byte, 1600) // UDP MTU
	for {
		n, _, err := listener.ReadFrom(inboundRTPPacket)
		if err != nil {
			panic(fmt.Sprintf("error during read: %s", err))
		}

		if _, err = videoTrack.Write(inboundRTPPacket[:n]); err != nil {
			if errors.Is(err, io.ErrClosedPipe) {
				// The peerConnection has been closed.
				log.Printf("PeerConnection Closed in rtp2webrtc: %s", clientId)
			}
			panic(err)
		}
	}
}

func ApplySDPAnswer(clientId string, sdpAnswer string) {
	fmt.Printf("RTP 2 WebRTC Answer from Client: %s --- \n %s \n --- \n", clientId, sdpAnswer)
	if clientPeerConnections[clientId] != nil {
		clientPeerConnections[clientId].SetRemoteDescription(webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  sdpAnswer,
		})
	} else {
		log.Printf("PeerConnection Not found for sdpAnswer in rtp2webrtc: %s", clientId)
	}
}
