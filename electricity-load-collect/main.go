package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"
	"math/rand" // 在这里加入随机数包

	"github.com/d2r2/go-shell"

	"github.com/yosssi/gmq/mqtt"
	"github.com/yosssi/gmq/mqtt/client"

	logger "github.com/d2r2/go-logger"
)

var lg = logger.NewPackageLogger("main",
	logger.DebugLevel,
	// logger.InfoLevel,
)

//DeviceStateUpdate is the structure used in updating the device state
type DeviceStateUpdate struct {
	State string `json:"state,omitempty"`
}

//BaseMessage the base struct of event message
type BaseMessage struct {
	EventID   string `json:"event_id"`
	Timestamp int64  `json:"timestamp"`
}

//TwinValue the struct of twin value
type TwinValue struct {
	Value    *string        `json:"value, omitempty"`
	Metadata *ValueMetadata `json:"metadata,omitempty"`
}

//ValueMetadata the meta of value
type ValueMetadata struct {
	Timestamp int64 `json:"timestamp, omitempty"`
}

//TypeMetadata the meta of value type
type TypeMetadata struct {
	Type string `json:"type,omitempty"`
}

//TwinVersion twin version
type TwinVersion struct {
	CloudVersion int64 `json:"cloud"`
	EdgeVersion  int64 `json:"edge"`
}

//MsgTwin the struct of device twin
type MsgTwin struct {
	Expected        *TwinValue    `json:"expected,omitempty"`
	Actual          *TwinValue    `json:"actual,omitempty"`
	Optional        *bool         `json:"optional,omitempty"`
	Metadata        *TypeMetadata `json:"metadata,omitempty"`
	ExpectedVersion *TwinVersion  `json:"expected_version,omitempty"`
	ActualVersion   *TwinVersion  `json:"actual_version,omitempty"`
}

//DeviceTwinUpdate the struct of device twin update
type DeviceTwinUpdate struct {
	BaseMessage
	Twin map[string]*MsgTwin `json:"twin"`
}

func main() {
	defer logger.FinalizeLogger()

	lg.Notify("***************************************************************************************************")
	lg.Notify("*** Uncomment/comment corresponding lines with call to ChangePackageLogLevel(...)")
	lg.Notify("***************************************************************************************************")
	lg.Notify("*** Massive stress test of sensor reading, printing in the end summary statistical results")
	lg.Notify("***************************************************************************************************")

	// create context with cancellation possibility
	ctx, cancel := context.WithCancel(context.Background())
	// use done channel as a trigger to exit from signal waiting goroutine
	done := make(chan struct{})
	defer close(done)
	// build actual signal list to control
	signals := []os.Signal{os.Kill, os.Interrupt}
	if shell.IsLinuxMacOSFreeBSD() {
		signals = append(signals, syscall.SIGTERM)
	}
	// run goroutine waiting for OS termination events, including keyboard Ctrl+C
	shell.CloseContextOnSignals(cancel, done, signals...)

	// sensorType := dht.DHT11
	// sensorType := dht.AM2302
	//sensorType := dht.DHT12
	// pin := 11
	totalRetried := 0
	totalMeasured := 0
	totalFailed := 0
	term := false

	// connect to Mqtt broker
	cli := connectToMqtt()
	rand.Seed(time.Now().Unix()) // 初始化随机数种子
	for {
		// Read DHT11 sensor data from specific pin, retrying 10 times in case of failure.
		// electricity, humidity, retried, err :=
		// 	dht.ReadDHTxxWithContextAndRetry(ctx, sensorType, pin, false, 10)
		
		electricity := float32(rand.Intn(100)) // 随机生成温度
		humidity := 0 // 赋固定值
		retried := 1 // 赋固定值
		var err error = nil // 赋固定值
		
		totalMeasured++
		totalRetried += retried
		if err != nil && ctx.Err() == nil {
			totalFailed++
			lg.Error(err)
			continue
		}
		// print electricity and humidity
		if ctx.Err() == nil {
			lg.Infof("electricity = %v*W, load = %v%% (retried %d times)",
				electricity, humidity, retried)
		}

		// publish electricity status to mqtt broker
		publishToMqtt(cli, electricity)

		select {
		// Check for termination request.
		case <-ctx.Done():
			lg.Errorf("Termination pending: %s", ctx.Err())
			term = true
			// sleep 1.5-2 sec before next round
			// (recommended by specification as "collecting period")
		case <-time.After(2000 * time.Millisecond):
		}
		if term {
			break
		}
	}
	lg.Info("exited")
}

func connectToMqtt() *client.Client {
	cli := client.New(&client.Options{
		// Define the processing of the error handler.
		ErrorHandler: func(err error) {
			fmt.Println(err)
		},
	})
	defer cli.Terminate()

	// Connect to the MQTT Server.
	err := cli.Connect(&client.ConnectOptions{
		Network:  "tcp",
		Address:  "127.0.0.1:1883",
		ClientID: []byte("receive-client"),
	})
	if err != nil {
		panic(err)
	}
	return cli
}

func publishToMqtt(cli *client.Client, electricity float32) {
	deviceTwinUpdate := "$hw/events/device/" + "electricity" + "/twin/update"

	updateMessage := createActualUpdateMessage(strconv.Itoa(int(electricity)) + "W")
	twinUpdateBody, _ := json.Marshal(updateMessage)

	cli.Publish(&client.PublishOptions{
		TopicName: []byte(deviceTwinUpdate),
		QoS:       mqtt.QoS0,
		Message:   twinUpdateBody,
	})
}

//createActualUpdateMessage function is used to create the device twin update message
func createActualUpdateMessage(actualValue string) DeviceTwinUpdate {
	var deviceTwinUpdateMessage DeviceTwinUpdate
	actualMap := map[string]*MsgTwin{"electricity-status": {Actual: &TwinValue{Value: &actualValue}, Metadata: &TypeMetadata{Type: "Updated"}}}
	deviceTwinUpdateMessage.Twin = actualMap
	return deviceTwinUpdateMessage
}
