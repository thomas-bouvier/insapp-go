package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	apns "github.com/anachronistic/apns"
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"net/http"
	"strings"
)

// fcmResponseStatus represents fcm response message
type fcmResponseStatus struct {
	Ok            bool
	StatusCode    int
	MulticastId   int64               `json:"multicast_id"`
	Success       int                 `json:"success"`
	Fail          int                 `json:"failure"`
	Canonical_ids int                 `json:"canonical_ids"`
	Results       []map[string]string `json:"results,omitempty"`
	MsgId         int64               `json:"message_id,omitempty"`
	Err           string              `json:"error,omitempty"`
	RetryAfter    string
}

func getiOSUsers(user string) []NotificationUser {
	conf, _ := Configuration()
	session, _ := mgo.Dial(conf.Database)
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	db := session.DB("insapp").C("notification_user")
	var result []NotificationUser
	if user == "" {
		db.Find(bson.M{"os": "iOS"}).All(&result)
	} else {
		db.Find(bson.M{"os": "iOS", "userid": user}).All(&result)
	}
	return result
}

func getAndroidUsers(user string) []NotificationUser {
	conf, _ := Configuration()
	session, _ := mgo.Dial(conf.Database)
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	db := session.DB("insapp").C("notification_user")
	var result []NotificationUser
	if user == "" {
		db.Find(bson.M{"os": "android"}).All(&result)
	} else {
		db.Find(bson.M{"os": "android", "userid": user}).All(&result)
	}
	return result
}

func getNotificationUserForUser(user bson.ObjectId) NotificationUser {
	conf, _ := Configuration()
	session, _ := mgo.Dial(conf.Database)
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	db := session.DB("insapp").C("notification_user")
	var result NotificationUser
	db.Find(bson.M{"userid": user}).One(&result)
	return result
}

func TriggerNotificationForUser(sender bson.ObjectId, receiver bson.ObjectId, content bson.ObjectId, message string, comment Comment, tagType string) {
	notification := Notification{Sender: sender, Content: content, Message: message, Comment: comment, Type: tagType}
	user := getNotificationUserForUser(receiver)
	if user.Os == "iOS" {
		triggeriOSNotification(notification, []NotificationUser{user})
	}
	if user.Os == "android" {
		triggerAndroidNotification(notification, []NotificationUser{user})
	}
}

func TriggerNotificationForEvent(event Event, sender bson.ObjectId, content bson.ObjectId, message string) {
	notification := Notification{Sender: sender, Content: content, Message: message, Type: "event"}
	iOSUsers := getiOSUsers("")
	users := []NotificationUser{}
	for _, notificationUser := range iOSUsers {
		var user = GetUser(notificationUser.UserId)
		if Contains(strings.ToUpper(user.Promotion), event.Promotions) {
			users = append(users, notificationUser)
		}
	}
	if Contains("iOS", event.Plateforms) {
		triggeriOSNotification(notification, users)
	}
	androidUsers := getAndroidUsers("")
	users = []NotificationUser{}
	for _, notificationUser := range androidUsers {
		var user = GetUser(notificationUser.UserId)
		if Contains(strings.ToUpper(user.Promotion), event.Promotions) {
			users = append(users, notificationUser)
		}
	}
	if Contains("android", event.Plateforms) {
		triggerAndroidNotification(notification, users)
	}
}

func TriggerNotificationForPost(post Post, sender bson.ObjectId, content bson.ObjectId, message string) {
	notification := Notification{Sender: sender, Content: content, Message: message, Type: "post"}
	iOSUsers := getiOSUsers("")
	users := []NotificationUser{}
	for _, notificationUser := range iOSUsers {
		var user = GetUser(notificationUser.UserId)
		if Contains(strings.ToUpper(user.Promotion), post.Promotions) {
			users = append(users, notificationUser)
		}
	}
	if Contains("iOS", post.Plateforms) {
		triggeriOSNotification(notification, users)
	}
	androidUsers := getAndroidUsers("")
	users = []NotificationUser{}
	for _, notificationUser := range androidUsers {
		var user = GetUser(notificationUser.UserId)
		if Contains(strings.ToUpper(user.Promotion), post.Promotions) {
			users = append(users, notificationUser)
		}
	}
	if Contains("android", post.Plateforms) {
		triggerAndroidNotification(notification, users)
	}
}

func triggerAndroidNotification(notification Notification, users []NotificationUser) {
	done := make(chan bool)
	for _, user := range users {
		notification.Receiver = user.UserId
		notification = AddNotification(notification)
		number := len(GetUnreadNotificationsForUser(user.UserId))
		go sendAndroidNotificationToDevice(user.Token, notification, number, done)
	}
	<-done
}

func triggeriOSNotification(notification Notification, users []NotificationUser) {
	done := make(chan bool)
	for _, user := range users {
		notification.Receiver = user.UserId
		notification = AddNotification(notification)
		number := len(GetUnreadNotificationsForUser(user.UserId))
		go sendiOSNotificationToDevice(user.Token, notification, number, done)
	}
	<-done
}

func sendiOSNotificationToDevice(token string, notification Notification, number int, done chan bool) {
	payload := apns.NewPayload()
	payload.Alert = notification.Message
	payload.Badge = number
	payload.Sound = "bingbong.aiff"

	pn := apns.NewPushNotification()
	pn.DeviceToken = token
	pn.AddPayload(payload)
	pn.Set("id", notification.ID)
	pn.Set("type", notification.Type)
	pn.Set("sender", notification.Sender)
	pn.Set("content", notification.Content)
	pn.Set("message", notification.Message)
	if notification.Type == "tag" {
		pn.Set("comment", notification.Comment.ID)
	}

	config, _ := Configuration()

	if config.Environment != "prod" {
		client := apns.NewClient("gateway.sandbox.push.apple.com:2195", "InsappDevCert.pem", "InsappDev.pem")
		client.Send(pn)
		pn.PayloadString()
	} else {
		client := apns.NewClient("gateway.push.apple.com:2195", "InsappProdCert.pem", "InsappProd.pem")
		client.Send(pn)
		pn.PayloadString()
	}

	done <- true
}

func sendAndroidNotificationToDevice(token string, notification Notification, number int, done chan bool) {
	url := "https://fcm.googleapis.com/fcm/send"
	notifJson, _ := json.Marshal(notification)

	var jsonStr string
	config, _ := Configuration()

	if config.Environment != "prod" {
		jsonStr = "{\"to\":\"" + token + "\", \"data\":" + string(notifJson) + ", \"restricted_package_name\":\"fr.insapp.insapp.debug\"}"
	} else {
		jsonStr = "{\"to\":\"" + token + "\", \"data\":" + string(notifJson) + ", \"restricted_package_name\":\"fr.insapp.insapp\"}"
	}

	req, _ := http.NewRequest("POST", url, bytes.NewBufferString(jsonStr))

	req.Header.Set("Authorization", "key="+config.GoogleKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)

	defer resp.Body.Close()

	fmt.Println("Android notification response :")
	fmt.Println("Token:", token)
	fmt.Println("Status:", resp.StatusCode)

	var res fcmResponseStatus

	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal([]byte(body), &res)

	spew.Dump(res)

	done <- true
}
