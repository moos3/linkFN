package main

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/mailgun/mailgun-go.v1"
)

// **** MAILGUN SETTINGS ****
// TODO: Move to ENV's in the OS or Container
// mailgun domain names
var yourDomain string = os.Getenv("MAILGUN_DOMAIN") //"mg.makerdev.nl" // e.g. mg.yourcompany.com

// starts with "key-"
var privateAPIKey string = os.Getenv("MAILGUN_PRIV_API_KEY") //"41ba7c4ab2eeae72e230e99ce31d445f-e44cc7c1-b556d011"

// starts with "pubkey-"
var publicValidationKey string = os.Getenv("MAILGUN_PUBLIC_VALID_KEY") //"pubkey-8e185f8d9740bd85d4e41d0bf6b7e510"

// Send messages to
var sendTo string = os.Getenv("MAIL_RECPT") //"richard.genthner@makerbot.com"

// Who the messages are from
var replyTo string = os.Getenv("MAIL_REPLY_TO") //"no-reply@makerbot.com"

// **** END MAILGUN SETTINGS *****
// triggerMessage -
// This used for sending the output via email and building Mailgun object
func triggerMessage(message string, link string) {
	// Create an instance of the Mailgun Client
	mg := mailgun.NewMailgun(yourDomain, privateAPIKey, publicValidationKey)

	sender := replyTo
	subject := "404 Detected: " + link
	body := message
	recipient := sendTo

	sendMessage(mg, sender, subject, body, recipient)
}

// sendMessage -
// Mailgun message sender
func sendMessage(mg mailgun.Mailgun, sender, subject, body, recipient string) {
	message := mg.NewMessage(sender, subject, body, recipient)
	resp, id, err := mg.Send(message)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ID: %s Resp: %s\n", id, resp)
}
