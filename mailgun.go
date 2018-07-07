package main

import (
	"fmt"
	"log"

	"gopkg.in/mailgun/mailgun-go.v1"
)

// **** MAILGUN SETTINGS ****
// TODO: Move to ENV's in the OS or Container
// mailgun domain names
var yourDomain = config.Mailgun.Domain //"mg.makerdev.nl" // e.g. mg.yourcompany.com

// starts with "key-"
var privateAPIKey = config.Mailgun.DomainAPIKey //"41ba7c4ab2eeae72e230e99ce31d445f-e44cc7c1-b556d011"

// starts with "pubkey-"
var publicValidationKey = config.Mailgun.PublicKey //"pubkey-8e185f8d9740bd85d4e41d0bf6b7e510"

var sendTo, replyTo string

// **** END MAILGUN SETTINGS *****
// triggerMessage -
// This used for sending the output via email and building Mailgun object
func triggerMessage(sendTo string, replyTo string, message string, link string) {
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
