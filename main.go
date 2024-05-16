package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/fiatjaf/eventstore/badger"
	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/policies"
	"github.com/nbd-wtf/go-nostr"

	"github.com/joho/godotenv"
)

func main() {
	relayPrivateKey, relayPublicKey := getRelayKeyPair()

	relay := initRelay()

	applyRelayInfo(relay, relayPublicKey)
	applyRelayDB(relay)
	applyRelayPolicies(relay)
	applyRelayRouters(relay, relayPublicKey, relayPrivateKey)

	fmt.Println("running on :3334")
	http.ListenAndServe(":3334", relay)
}

func getRelayKeyPair() (string, string) {
	err := godotenv.Load()
	if err != nil {
    log.Fatal("Error loading .env file")
	}

	relayPrivateKey := os.Getenv("RELAY_PRIVATE_KEY")
	relayPublicKey := os.Getenv("RELAY_PUBLIC_KEY")
	if relayPrivateKey == "" || relayPublicKey == "" {
		log.Fatal("RELAY_PRIVATE_KEY and RELAY_PUBLIC_KEY must be set in .env")
	}

	return relayPrivateKey, relayPublicKey
}

func initRelay() *khatru.Relay {
	return khatru.NewRelay()
}

func applyRelayInfo(relay *khatru.Relay, relayPublicKey string) {
	relay.Info.Name = "Chatstr Relay"
	relay.Info.PubKey = relayPublicKey
	relay.Info.SupportedNIPs = []int{29}
	relay.Info.Description = "NIP29 relay for Chatstr"	
}

func applyRelayDB(relay *khatru.Relay) {
	db := badger.BadgerBackend{Path: "/tmp/khatru-badgern-tmp"}
	if err := db.Init(); err != nil {
		panic(err)
	}

	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)
	relay.CountEvents = append(relay.CountEvents, db.CountEvents)
	relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent)
}

func applyRelayPolicies(relay *khatru.Relay) {
	// First Apply Sane Default Policies
	policies.ApplySaneDefaults(relay)

	// Then Restrict To Specified Kinds (NIP29)
	nip29AllKinds := []uint16{9, 10, 11, 12, 39000, 39001, 39002}
	for i := uint16(9000); i <= 9021; i++ {
		nip29AllKinds = append(nip29AllKinds, i)
	}
	policies.RestrictToSpecifiedKinds(nip29AllKinds...)
}

func applyRelayRouters(relay *khatru.Relay, relayPublicKey string, relayPrivateKey string) {
	mux := relay.Router()

	// Home Page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html")
		fmt.Fprint(w, `<b>welcome</b> to my relay!<br />`)
	})

	mux.HandleFunc("/t", func(w http.ResponseWriter, r *http.Request) {
		// Create a new event
		ev := nostr.Event{
			PubKey:    relayPublicKey,
			CreatedAt: nostr.Now(),
			Kind:      nostr.KindTextNote,
			Tags:      nil,
			Content:   "Hello World!",
		}

		// Sign the event with the relay's private key
		ev.Sign(relayPrivateKey)

		// Add the event to the relay and publish to all subscribers
		ctx := context.Background()
		relay.AddEvent(ctx, &ev)

		w.Header().Set("content-type", "text/html")
		fmt.Fprint(w, `ok!`)
	})
}