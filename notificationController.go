package insapp

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2/bson"
)

func UpdateNotificationUserController(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var user NotificationUser
	decoder.Decode(&user)

	CreateOrUpdateNotificationUser(user)

	json.NewEncoder(w).Encode(bson.M{"status": "ok"})
}

func GetNotificationController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	res := GetNotificationsForUser(bson.ObjectIdHex(userID))

	json.NewEncoder(w).Encode(bson.M{"notifications": res})
}

func DeleteNotificationController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]
	notificationID := vars["id"]

	res := ReadNotificationForUser(bson.ObjectIdHex(userID), bson.ObjectIdHex(notificationID))

	json.NewEncoder(w).Encode(bson.M{"notifications": res})
}
