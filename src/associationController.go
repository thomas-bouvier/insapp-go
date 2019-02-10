package main

import (
	"encoding/json"
	"github.com/freehaha/token-auth"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2/bson"
	"net/http"
)

func GetMyAssociationController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	associationID := vars["id"]
	var res = GetMyAssociations(bson.ObjectIdHex(associationID))
	_ = json.NewEncoder(w).Encode(res)
}

// GetAssociationController will answer a JSON of the association
// linked to the given id in the URL
func GetAssociationController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	associationID := vars["id"]
	var res = GetAssociation(bson.ObjectIdHex(associationID))
	_ = json.NewEncoder(w).Encode(res)
}

// GetAllAssociationsController will answer a JSON of all associations
func GetAllAssociationsController(w http.ResponseWriter, r *http.Request) {
	var res = GetAllAssociation()
	_ = json.NewEncoder(w).Encode(res)
}

func CreateUserForAssociationController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	associationID := vars["id"]
	var res = GetAssociation(bson.ObjectIdHex(associationID))

	decoder := json.NewDecoder(r.Body)
	var user AssociationUser
	_ = decoder.Decode(&user)

	isValid := VerifyAssociationRequest(r, bson.ObjectIdHex(associationID))
	if !isValid {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(bson.M{"error": "protected content"})
		return
	}

	password := GeneratePassword()
	user.Association = res.ID
	user.Username = res.Email
	user.Password = GetMD5Hash(password)
	AddAssociationUser(user)
	_ = SendAssociationEmailSubscription(user.Username, password)
	_ = json.NewEncoder(w).Encode(res)
}

// AddAssociationController will answer a JSON of the
// brand new created association (from the JSON Body)
func AddAssociationController(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var association Association
	_ = decoder.Decode(&association)
	isValid := VerifyAssociationRequest(r, association.ID)
	if !isValid {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(bson.M{"error": "protected content"})
		return
	}
	res := AddAssociation(association)
	password := GeneratePassword()
	token := tauth.Get(r)
	id := bson.ObjectIdHex(token.Claims("id").(string))

	var user AssociationUser
	user.Association = res.ID
	user.Username = res.Email
	user.Master = false
	user.Owner = id
	user.Password = GetMD5Hash(password)
	AddAssociationUser(user)
	_ = SendAssociationEmailSubscription(user.Username, password)
	_ = json.NewEncoder(w).Encode(res)
}

// UpdateAssociationController will answer the JSON of the
// modified association (from the JSON Body)
func UpdateAssociationController(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var association Association
	_ = decoder.Decode(&association)
	vars := mux.Vars(r)
	associationID := vars["id"]
	isValid := VerifyAssociationRequest(r, bson.ObjectIdHex(associationID))
	if !isValid {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(bson.M{"error": "protected content"})
		return
	}
	res := UpdateAssociation(bson.ObjectIdHex(associationID), association)
	_ = json.NewEncoder(w).Encode(res)
}

// DeleteAssociationController will answer a JSON of an
// empty association if the deletion has succeed
func DeleteAssociationController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	associationID := vars["id"]
	isValid := VerifyAssociationRequest(r, bson.ObjectIdHex(associationID))
	if !isValid {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(bson.M{"error": "protected content"})
		return
	}
	res := DeleteAssociation(bson.ObjectIdHex(associationID))
	_ = json.NewEncoder(w).Encode(res)
}

func VerifyAssociationRequest(r *http.Request, associationId bson.ObjectId) bool {
	token := tauth.Get(r)
	id := token.Claims("id").(string)
	if bson.ObjectIdHex(id) != associationId {
		result := GetAssociationUser(bson.ObjectIdHex(id))
		return result.Master
	}
	return true
}
