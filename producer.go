package mediasoup

import (
	"encoding/json"
	"sync"
	"sync/atomic"
)

type ProducerOptions struct {
	/**
	 * Producer id (just for Router.pipeToRouter() method).
	 */
	Id string

	/**
	 * Media kind ('audio' or 'video').
	 */
	Kind MediaKind

	/**
	 * RTP parameters defining what the endpoint is sending.
	 */
	RtpParameters RtpParameters

	/**
	 * Whether the producer must start in paused mode. Default false.
	 */
	Paused bool

	/**
	 * Just for video. Time (in ms) before asking the sender for a new key frame
	 * after having asked a previous one. Default 0.
	 */
	KeyFrameRequestDelay uint32

	/**
	 * Custom application data.
	 */
	AppData interface{}
}

/**
 * Valid types for 'trace' event.
 */
type ProducerTraceEventType string

const (
	ProducerTraceEventType_Rtp      ProducerTraceEventType = "rtp"
	ProducerTraceEventType_Keyframe                        = "keyframe"
	ProducerTraceEventType_Nack                            = "nack"
	ProducerTraceEventType_Pli                             = "pli"
	ProducerTraceEventType_Fir                             = "fir"
)

/**
 * 'trace' event data.
 */
type ProducerTraceEventData struct {
	/**
	 * Trace type.
	 */
	Type ProducerTraceEventType `json:"type,omitempty"`

	/**
	 * Event timestamp.
	 */
	Timestamp uint32 `json:"timestamp,omitempty"`

	/**
	 * Event direction, "in" | "out".
	 */
	Direction string `json:"direction,omitempty"`

	/**
	 * Per type information.
	 */
	Info H `json:"info,omitempty"`
}

type ProducerScore struct {
	/**
	 * SSRC of the RTP stream.
	 */
	Ssrc uint32 `json:"ssrc,omitempty"`

	/**
	 * RID of the RTP stream.
	 */
	Rid string `json:"rid,omitempty"`

	/**
	 * The score of the RTP stream.
	 */
	Score uint32 `json:"score,omitempty"`
}

type ProducerVideoOrientation struct {
	/**
	 * Whether the source is a video camera.
	 */
	Camera bool `json:"Camera,omitempty"`

	/**
	 * Whether the video source is flipped.
	 */
	Flip bool `json:"flip,omitempty"`

	/**
	 * Rotation degrees (0, 90, 180 or 270).
	 */
	Rotation uint32 `json:"rotation,omitempty"`
}

type ProducerStat struct {
	// Common to all RtpStreams.
	Type                 string `json:"type"`
	Timestamp            uint32 `json:"timestamp"`
	Ssrc                 uint32 `json:"ssrc"`
	RtxSsrc              uint32 `json:"rtxSsrc,omitempty"`
	Rid                  string `json:"rid,omitempty"`
	Kind                 string `json:"kind"`
	MimeType             string `json:"mimeType"`
	PacketsLost          uint32 `json:"packetsLost"`
	FractionLost         uint32 `json:"fractionLost"`
	PacketsDiscarded     uint32 `json:"packetsDiscarded"`
	PacketsRetransmitted uint32 `json:"packetsRetransmitted"`
	PacketsRepaired      uint32 `json:"packetsRepaired"`
	NackCount            uint32 `json:"nackCount"`
	NackPacketCount      uint32 `json:"nackPacketCount"`
	PliCount             uint32 `json:"pliCount"`
	FirCount             uint32 `json:"firCount"`
	Score                uint32 `json:"score"`
	PacketCount          uint32 `json:"packetCount"`
	ByteCount            uint32 `json:"byteCount"`
	Bitrate              uint32 `json:"bitrate"`
	RoundTripTime        uint32 `json:"roundTripTime,omitempty"`

	// RtpStreamRecv specific.
	Jitter         uint32 `json:"jitter,omitempty"`
	BitrateByLayer H      `json:"bitrateByLayer,omitempty"`
}

/**
 * Producer type.
 */
type ProducerType string

const (
	ProducerType_Simple    ProducerType = "simple"
	ProducerType_Simulcast              = "simulcast"
	ProducerType_Svc                    = "svc"
)

type producerData struct {
	Kind                    MediaKind     `json:"kind,omitempty"`
	Type                    ProducerType  `json:"type,omitempty"`
	RtpParameters           RtpParameters `json:"rtpParameters,omitempty"`
	ConsumableRtpParameters RtpParameters `json:"consumableRtpParameters,omitempty"`
}

type newProducerOptions struct {
	// Internal data.
	// {
	// 	 routerId: string;
	// 	 transportId: string;
	// 	 producerId: string;
	// };
	internal       internalData
	data           producerData
	channel        *Channel
	payloadChannel *PayloadChannel
	appData        interface{}
	paused         bool
}

type Producer struct {
	IEventEmitter
	locker         sync.Mutex
	logger         Logger
	internal       internalData
	data           producerData
	channel        *Channel
	payloadChannel *PayloadChannel
	appData        interface{}
	paused         bool
	closed         uint32
	score          []ProducerScore
	observer       IEventEmitter
}

/**
 * newProducer
 *
 * @emits transportclose
 * @emits score - (score: ProducerScore[])
 * @emits videoorientationchange - (videoOrientation: ProducerVideoOrientation)
 * @emits trace - (trace: ProducerTraceEventData)
 * @emits @close
 */
func newProducer(options newProducerOptions) *Producer {
	logger := NewLogger("Producer")

	logger.Debug("constructor()")

	producer := &Producer{
		IEventEmitter:  NewEventEmitter(),
		logger:         logger,
		internal:       options.internal,
		data:           options.data,
		channel:        options.channel,
		payloadChannel: options.payloadChannel,
		appData:        options.appData,
		paused:         options.paused,
		observer:       NewEventEmitter(),
	}

	producer.handleWorkerNotifications()

	return producer
}

// Producer id
func (producer *Producer) Id() string {
	return producer.Id()
}

// Whether the Producer is closed.
func (producer *Producer) Closed() bool {
	return atomic.LoadUint32(&producer.closed) > 0
}

// Media kind.
func (producer *Producer) Kind() MediaKind {
	return producer.data.Kind
}

// RTP parameters.
func (producer *Producer) RtpParameters() RtpParameters {
	return producer.data.RtpParameters
}

// Producer type.
func (producer *Producer) Type() ProducerType {
	return producer.data.Type
}

// Consumable RTP parameters.
func (producer *Producer) ConsumableRtpParameters() RtpParameters {
	return producer.data.ConsumableRtpParameters
}

// Whether the Producer is paused.
func (producer *Producer) Paused() bool {
	producer.locker.Lock()
	defer producer.locker.Unlock()

	return producer.paused
}

// Producer score list.
func (producer *Producer) Score() []ProducerScore {
	return producer.score
}

//App custom data.
func (producer *Producer) AppData() interface{} {
	return producer.appData
}

/**
 * Observer.
 *
 * @emits close
 * @emits pause
 * @emits resume
 * @emits score - (score: ProducerScore[])
 * @emits videoorientationchange - (videoOrientation: ProducerVideoOrientation)
 * @emits trace - (trace: ProducerTraceEventData)
 */
func (producer *Producer) Observer() IEventEmitter {
	return producer.observer
}

// Close the Producer.
func (producer *Producer) Close() (err error) {
	if atomic.CompareAndSwapUint32(&producer.closed, 0, 1) {
		producer.logger.Debug("close()")

		// Remove notification subscriptions.
		producer.channel.RemoveAllListeners(producer.Id())
		producer.payloadChannel.RemoveAllListeners(producer.Id())

		response := producer.channel.Request("producer.close", producer.internal)

		if err = response.Err(); err != nil {
			return
		}

		producer.Emit("@close")

		// Emit observer event.
		producer.observer.SafeEmit("close")
	}

	return
}

// Transport was closed.
func (producer *Producer) transportClosed() {
	if atomic.CompareAndSwapUint32(&producer.closed, 0, 1) {
		producer.logger.Debug("transportClosed()")

		// Remove notification subscriptions.
		producer.channel.RemoveAllListeners(producer.Id())
		producer.payloadChannel.RemoveAllListeners(producer.Id())

		producer.SafeEmit("transportclose")

		// Emit observer event.
		producer.observer.SafeEmit("close")
	}
}

// Dump Producer.
func (producer *Producer) Dump() ([]byte, error) {
	producer.logger.Debug("dump()")

	resp := producer.channel.Request("producer.dump", producer.internal)

	return resp.Data(), resp.Err()
}

// Get Producer stats.
func (producer *Producer) GetStats() (stats []ProducerStat, err error) {
	producer.logger.Debug("getStats()")

	resp := producer.channel.Request("producer.getStats", producer.internal)
	err = resp.Unmarshal(&stats)

	return
}

// Pause the Producer.
func (producer *Producer) Pause() (err error) {
	producer.locker.Lock()
	defer producer.locker.Unlock()

	producer.logger.Debug("pause()")

	wasPaused := producer.paused

	response := producer.channel.Request("producer.pause", producer.internal)

	if err = response.Err(); err != nil {
		return
	}

	producer.paused = true

	// Emit observer event.
	if !wasPaused {
		producer.observer.SafeEmit("pause")
	}

	return
}

// Resume the Producer.
func (producer *Producer) Resume() (err error) {
	producer.locker.Lock()
	defer producer.locker.Unlock()

	producer.logger.Debug("resume()")

	wasPaused := producer.paused

	result := producer.channel.Request("producer.resume", producer.internal)

	if err = result.Err(); err != nil {
		return
	}

	producer.paused = false

	// Emit observer event.
	if wasPaused {
		producer.observer.SafeEmit("resume")
	}

	return
}

/**
 * Enable 'trace' event.
 */
func (producer *Producer) EnableTraceEvent(types ...ProducerTraceEventType) error {
	producer.logger.Debug("enableTraceEvent()")

	result := producer.channel.Request("producer.enableTraceEvent", producer.internal, H{"types": types})

	return result.Err()
}

/**
 * Send RTP packet (just valid for Producers created on a DirectTransport).
 */
func (producer *Producer) Send(rtpPacket []byte) error {
	result := producer.payloadChannel.Request("producer.send", producer.internal, nil, rtpPacket)

	return result.Err()
}

func (producer *Producer) handleWorkerNotifications() {
	producer.channel.On(producer.Id(), func(event string, data []byte) {
		switch event {
		case "score":
			producer.score = []ProducerScore{}

			json.Unmarshal([]byte(data), &producer.score)

			producer.SafeEmit("score", producer.score)

			// Emit observer event.
			producer.observer.SafeEmit("score", producer.score)

		case "videoorientationchange":
			orientation := ProducerVideoOrientation{}

			json.Unmarshal([]byte(data), &orientation)

			producer.SafeEmit("videoorientationchange", orientation)

			// Emit observer event.
			producer.observer.SafeEmit("videoorientationchange", orientation)

		case "trace":
			var trace ProducerTraceEventData

			json.Unmarshal(data, &trace)

			producer.SafeEmit("trace", trace)

			// Emit observer event.
			producer.observer.SafeEmit("trace", trace)

		default:
			producer.logger.Error(`ignoring unknown event "%s"`, event)
		}
	})
}