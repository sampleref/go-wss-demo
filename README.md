# go-wss-demo
Simple demo to validate webrtc on browser via whip

# Example WHIP Source from GStreamer(Audio + Video)
`gst-launch-1.0 videotestsrc ! videoconvert ! openh264enc ! rtph264pay ! \
'application/x-rtp,media=video,encoding-name=H264,payload=97,clock-rate=90000' ! \
whip.sink_0 audiotestsrc ! audioconvert ! opusenc ! rtpopuspay ! \
'application/x-rtp,media=audio,encoding-name=OPUS,payload=96,clock-rate=48000,encoding-params=(string)2' ! \
whip.sink_1 whipsink name=whip whip-endpoint="http://localhost:8080/whip?clientId=1001"`

# Example WHIP Source from GStreamer(Video Only)
`gst-launch-1.0 autovideosrc ! videoconvert ! openh264enc ! rtph264pay ! \
'application/x-rtp,media=video,encoding-name=H264,payload=97,clock-rate=90000' ! \
whip.sink_0 whipsink name=whip stun-server=stun://stun.l.google.com:19302 whip-endpoint="http://localhost:8081/whip?clientId=3733"`

## Steps
1. Run `./gen_cert.sh` to generate local servet.crt and server.key
2. Run `go build`
3. Run `./go-wss-demo`
4. Open URL `https://localhost:8080/` in browser/chrome
   skip/continue if any security exception due to unverified cert, 
   since this is only for local testing
5. Note down `clientId:{}` on webpage, Example: `ClientId: 2685`
6. Run above GStreamer command which sends video to WHIP endpoint via webrtc, apply above noted clientId in 
   GStreamer whipsink property whip-endpoint
   Example:
   `gst-launch-1.0 autovideosrc ! videoconvert ! openh264enc ! rtph264pay ! \
   'application/x-rtp,media=video,encoding-name=H264,payload=97,clock-rate=90000' ! \
   whip.sink_0 whipsink name=whip stun-server=stun://stun.l.google.com:19302 whip-endpoint="http://localhost:8081/whip?clientId=2685"`
