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
	fmt.Println("Starting Peril client...")

	conn, err := amqp.Dial(conn_str)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer conn.Close()

	fmt.Println("Connected to AMQP server")

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("could not create channel: %v", err)
	}

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal(err)
	}

	gs := gamelogic.NewGameState(username)

	_, err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, username),
		routing.PauseKey,
		pubsub.Transient,
		handlerPause(gs),
	)
	if err != nil {
		log.Fatal(err)
	}

	am_ch, err := pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username),
		fmt.Sprintf("%s.*", routing.ArmyMovesPrefix),
		pubsub.Transient,
		handlerMove(gs, publishCh),
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix+".*",
		pubsub.Durable,
		handlerWar(gs),
	)
	if err != nil {
		log.Fatalf("could not subscribe to war declarations: %v", err)
	}

repl:
	for {
		in := gamelogic.GetInput()
		switch in[0] {
		case "quit":
			gamelogic.PrintQuit()
			conn.Close()
			break repl

		case "spawn":
			err = gs.CommandSpawn(in)
			if err != nil {
				fmt.Println(err)
			}

		case "move":
			am, err := gs.CommandMove(in)
			if err != nil {
				fmt.Println(err)
				continue
			}
			err = pubsub.PublishJSON(am_ch, string(routing.ExchangePerilTopic), fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username), am)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println("Move sent successfully")

		case "status":
			gs.CommandStatus()

		case "help":
			gamelogic.PrintClientHelp()

		case "spam":
			fmt.Println("Spamming not allowed yet!")

		default:
			fmt.Printf("Unrecognized commmand: %s\n", in[0])
		}
	}

	fmt.Println("Shutdown complete.")
}
