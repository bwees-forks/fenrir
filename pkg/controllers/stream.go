package controllers

// Labels on every game pod identifying its App and User.
const (
	AppLabel  = "direwolf/app"
	UserLabel = "direwolf/user"
)

// StreamLabel selects the pods behind the Helm chart's stream Service.
const (
	StreamLabel      = "direwolf.games-on-whales.github.io/stream"
	StreamLabelValue = "true"
)
