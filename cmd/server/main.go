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

	_, _, err = pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		"game_logs",
		"game_logs.*",
		pubsub.Durable,
	)
	if err != nil {
		log.Fatal(err)
	}

	gamelogic.PrintServerHelp()

repl:
	for {
		in := gamelogic.GetInput()[0]
		switch in {
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
