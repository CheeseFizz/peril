package main

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	handler := func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}

	return handler
}

func handlerMove(gs *gamelogic.GameState, publishCh *amqp.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	handler := func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(move)

		switch outcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack

		case gamelogic.MoveOutcomeMakeWar:
			routeKey := fmt.Sprintf("%s.%s",
				routing.WarRecognitionsPrefix,
				gs.Player.Username,
			)
			msg := gamelogic.RecognitionOfWar{
				Attacker: move.Player,
				Defender: gs.GetPlayerSnap(),
			}
			err := pubsub.PublishJSON(publishCh, routing.ExchangePerilTopic, routeKey, msg)
			if err != nil {
				fmt.Printf("error: %s\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack

		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard

		default:
			return pubsub.NackDiscard

		}
	}

	return handler
}

func handlerWar(gs *gamelogic.GameState, publishCh *amqp.Channel) func(dw gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(dw gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		warOutcome, winner, loser := gs.HandleWar(dw)
		switch warOutcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue

		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard

		case gamelogic.WarOutcomeOpponentWon:
			msg := fmt.Sprintf("%s won a war against %s\n", winner, loser)
			err := gamelogic.PublishLog(publishCh, dw.Attacker.Username, msg)
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack

		case gamelogic.WarOutcomeYouWon:
			msg := fmt.Sprintf("%s won a war against %s\n", winner, loser)
			err := gamelogic.PublishLog(publishCh, dw.Attacker.Username, msg)
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack

		case gamelogic.WarOutcomeDraw:
			msg := fmt.Sprintf("A war between %s and %s resulted in a draw\n", winner, loser)
			err := gamelogic.PublishLog(publishCh, dw.Attacker.Username, msg)
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		}

		fmt.Println("error: unknown war outcome")
		return pubsub.NackDiscard
	}
}
