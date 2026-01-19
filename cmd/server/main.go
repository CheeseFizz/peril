package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	conn_str := "amqp://guest:guest@localhost:5672/"
	fmt.Println("Starting Peril server...")

	conn, err := amqp.Dial(conn_str)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer conn.Close()

	fmt.Println("Connected to AMQP server")

	mq_chan, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	err = pubsub.SubscribeGob(
		conn,
		routing.ExchangePerilTopic,
		string(routing.GameLogSlug),
		fmt.Sprintf("%s.*", routing.GameLogSlug),
		pubsub.Durable,
		handlerLog(),
	)
	if err != nil {
		log.Fatal(err)
	}

	gamelogic.PrintServerHelp()

repl:
	for {
		in := gamelogic.GetInput()
		if len(in) == 0 {
			continue
		}
		switch in[0] {
		case "pause":
			log.Println("Sending 'pause' message")
			data := routing.PlayingState{
				IsPaused: true,
			}
			err = pubsub.PublishJSON(mq_chan, routing.ExchangePerilDirect, routing.PauseKey, data)
			if err != nil {
				log.Fatal(err)
			}

		case "resume":
			log.Println("Sending 'resume' message")
			data := routing.PlayingState{
				IsPaused: false,
			}
			err = pubsub.PublishJSON(mq_chan, routing.ExchangePerilDirect, routing.PauseKey, data)
			if err != nil {
				log.Fatal(err)
			}

		case "quit":
			log.Println("Exiting...")
			conn.Close()
			break repl

		default:
			log.Printf("Unexpected keyword: %s\n", in)
		}
	}

	fmt.Println("Shutdown complete")
}
