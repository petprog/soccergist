package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"

	"github.com/julienschmidt/httprouter"
)

// VerificationToken is the random string entered in the verification prompt when setting up the app on Facbook
// It can be any string provide it matches what you will enter in the setup prompt
const VerificationToken = "bots are awesome"

var AccessToken = os.Getenv("ACCESS_TOKEN")

func verifyWebhook(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token == VerificationToken {
		fmt.Fprint(w, challenge)
	} else {
		w.WriteHeader(http.StatusForbidden)
	}
}

func handleWebhookEvents(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	io.Copy(os.Stdout, r.Body)
	// Parse the request payload
	payload := webhookPayload{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Println("Unmarshalling webhook payload resulted in an error: ", err)
		return
	}

	// Make sure this is a page subscription
	if payload.Object == "page" {
		// Iterate over each entry
		// There may be multiple if batched
		for _, entry := range payload.Entry {
			// Iterate over each messaging event
			for _, messaging := range entry.Messaging {
				switch {
				case !reflect.DeepEqual(messaging.Message, messageEvent{}):
					handleMessageEvent(messaging.Message, messaging.Sender.ID)
				default:
					log.Printf("No handler found for: %+v\n", messaging.Message)
				}
			}
		}
	}
}

func handleMessageEvent(msgEvnt messageEvent, senderID string) {
	reply := textResponse{}
	reply.Recipient.ID = senderID
	reply.Message.Text = fmt.Sprintf("I received your message: '%s', and I've sent it to my Oga at the top Oscar", msgEvnt.Text)
	sendResponse(reply)
}

func sendResponse(payload interface{}) {
	// Parse the response payload
	pkg, err := json.Marshal(payload)
	if err != nil {
		log.Println("Sending response parsing in an error: ", err)
		return
	}
	body := bytes.NewBuffer(pkg)

	fbURL := "https://graph.facebook.com/v2.6/me/messages?"
	url := fmt.Sprintf("%saccess_token=%s", fbURL, AccessToken)

	req, err := http.NewRequest("POST", url, body)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}

	_, err = client.Do(req)
	if err != nil {
		log.Println("Sending response resulted in an error: ", err)
	}
}

func setupRouter() *httprouter.Router {
	r := httprouter.New()
	r.GET("/webhook", verifyWebhook)
	r.POST("/webhook", handleWebhookEvents)
	return r
}

func main() {
	err := http.ListenAndServe(":3000", setupRouter())
	if err != nil {
		panic(err)
	}
}