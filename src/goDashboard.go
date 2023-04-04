package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	"github.com/boltdb/bolt"
	"github.com/spf13/viper"

	"solace.dev/go/messaging"
	"solace.dev/go/messaging/pkg/solace/config"
	"solace.dev/go/messaging/pkg/solace/message"
	"solace.dev/go/messaging/pkg/solace/resource"
)

//TODO replace c with cursor

// Some global variables because I am a lazy programmer
var (
	W *astilectron.Window
)

func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func btoi(v []byte) int {
	i := binary.BigEndian.Uint64(v)
	return int(i)
}

func MessageHandler(message message.InboundMessage) {
	fmt.Print("Message Dump %s \n", message)

	// Open bolt DB
	db, dberr := bolt.Open("solace.db", 0600, nil)
	if dberr != nil {
		log.Fatal(dberr)
	}
	err := db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("eventBucket"))
		if err != nil {
			log.Fatal(err)
		}
		id, _ := b.NextSequence()
		key := int(id)
		data, derr := message.GetPayloadAsBytes()
		if !derr {
			log.Fatal(derr)
		}

		fmt.Println("Wrote message to database at key: ", key)
		return b.Put(itob(key), data)
	})

	if err != nil {
		log.Fatal(err)
	}
	db.Close()

	//UpdateDisplay()
	ReadDatabase()
}

func ReadDatabase() {
	var number []byte
	var contents []byte
	var data [5][2]string

	// Open bolt DB
	db, dberr := bolt.Open("solace.db", 0600, nil)
	if dberr != nil {
		log.Fatal(dberr)
	}
	defer db.Close()
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("eventBucket"))
		if err != nil {
			log.Fatal("Failed to create bucket", err)
		}
		return nil
	})
	// Get 5 rows from the display table
	db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("eventBucket")).Cursor()

		if c != nil {
			// event range
			//min := itob(0)

			// iterate through table
			number, contents = c.First()

			if number != nil {
				data[0][0] = strconv.Itoa(btoi(number))
				data[0][1] = string(contents)
				//PushDisplayUpdate(number, contents)
				PushDisplay(data)
			}

			for count := 0; count < 4; count++ {
				number, contents = c.Next()
				if number != nil {
					data[count+1][0] = strconv.Itoa(btoi(number))
					data[count+1][1] = string(contents)
					//PushDisplayUpdate(number, contents)
					PushDisplay(data)
				}

			}
		} else {
			fmt.Println("No bucket for events exist yet, will be created when event is reciebed.")

		}
		return nil
	})
	db.Close()
}

func RemoveRow(row int) {
	// Open bolt DB
	db, dberr := bolt.Open("solace.db", 0600, nil)
	if dberr != nil {
		log.Fatal(dberr)
	}
	defer db.Close()

	// add the row to the display table
	db.Update(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("eventBucket")).Cursor()

		// iterate through table
		k, _ := c.Seek(itob(row))
		if k != nil {
			result := c.Delete()
			log.Println("Removed row: ", row, result)
		}

		return nil
	})
	db.Close()
	//UpdateDisplay()
	ReadDatabase()
}

func PushDisplay(data [5][2]string) {
	err := W.SendMessage(data, func(m *astilectron.EventMessage) {
		log.Println("Calling javascript")
	})
	if err != nil {
		log.Fatal(fmt.Errorf("UpdateDisplay: changing html: %w", err))
	}
}

func PopulateWindow() {

}

func BuildWindow() {
	var a, aerr = astilectron.New(log.New(os.Stderr, "", 0), astilectron.Options{
		AppName:           "Test Dashboard",
		BaseDirectoryPath: "./dashboard",
	})

	if aerr != nil {
		log.Fatal(fmt.Errorf("buildWindow: creating astilectron failed: %w", aerr))
	}

	defer a.Close()
	a.HandleSignals()

	if aerr = a.Start(); aerr != nil {
		log.Fatal(fmt.Errorf("buildWindow: starting astilectron failed:  %w", aerr))
	}

	// Build a new window in our app

	if W, aerr = a.NewWindow("./src/index.html", &astilectron.WindowOptions{
		Center: astikit.BoolPtr(true),
		Height: astikit.IntPtr(700),
		Width:  astikit.IntPtr(700),
	}); aerr != nil {
		log.Fatal(fmt.Errorf("buildWindow: new window failed: %w", aerr))
	}

	// Create window
	if aerr = W.Create(); aerr != nil {
		log.Fatal(fmt.Errorf("buildWindow: creating window failed: %w", aerr))
	}
	W.OpenDevTools()

	W.OnMessage(func(m *astilectron.EventMessage) interface{} {
		// Unmarshal
		var row int
		m.Unmarshal(&row)
		log.Println("Remove row: ", row)
		RemoveRow(row)
		return nil
	})
	ReadDatabase()
	a.Wait()
}

func main() {

	// load config values
	viper.SetConfigName("config")
	viper.AddConfigPath("./")
	viper.AutomaticEnv()
	viper.SetConfigType("yml")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	go BuildWindow()
	fmt.Println("Host: ", viper.GetString("Host"))

	TOPIC_PREFIX := "go/samples"
	messagingService, _ := messaging.NewMessagingServiceBuilder().FromConfigurationProvider(config.ServicePropertyMap{
		config.TransportLayerPropertyHost:                viper.GetString("Host"),
		config.ServicePropertyVPNName:                    viper.GetString("Vpn"),
		config.AuthenticationPropertySchemeBasicUserName: viper.GetString("Username"),
		config.AuthenticationPropertySchemeBasicPassword: viper.GetString("Password"),
	}).Build()

	err = messagingService.Connect()

	if err != nil {
		panic(err)
	}

	fmt.Println("Connected")

	// Define topic subscriptions
	topics := [...]string{TOPIC_PREFIX + "/account"}
	topics_sup := make([]resource.Subscription, len(topics))

	for i, topicString := range topics {
		topics_sup[i] = resource.TopicSubscriptionOf(topicString)
	}

	for _, ts := range topics_sup {
		fmt.Println("Subscribed to: ", ts.GetName())
	}

	// Build a direct message reciever for the topics
	directReceiver, err := messagingService.CreateDirectMessageReceiverBuilder().WithSubscriptions(topics_sup...).Build()

	if err != nil {
		panic(err)
	}

	// Start the reciever
	if err := directReceiver.Start(); err != nil {
		panic(err)
	}

	fmt.Println("Direct Receiver running: ", directReceiver.IsRunning())

	// Register message callback handler to the Message Receiver
	if regErr := directReceiver.ReceiveAsync(MessageHandler); regErr != nil {
		panic(regErr)
	}

	fmt.Println("\n===Interrupt (Ctr-c) to start graceful termination of this subscriber===\n")

	// Cleanup
	defer func() {
		// Terminate the Direct Receiver
		directReceiver.Terminate(1 * time.Second)
		fmt.Println("\nDirect Receiver terminated: ", directReceiver.IsTerminated())
		// Disconnect the message service
		messagingService.Disconnect()
		fmt.Println("Messaging Service Disconnected: ", !messagingService.IsConnected())
	}()

	// Loop forever
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	// Block until interupt is recieved
	<-c
}
