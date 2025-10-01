package domain

import "github.com/cskr/pubsub"

type Context struct {
	Hub *pubsub.PubSub
	Config
}
