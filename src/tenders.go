package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Tender struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	ServiceType     string    `json:"serviceType"`
	Status          string    `json:"status"`
	OrganizationID  uuid.UUID `json:"organizationId"`
	CreatorUsername string    `json:"creatorUsername"`
	Version         int       `json:"version"`
	CreatedAt       time.Time `json:"createdAt"`
}

func CreateTenderHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("CreateTenderHandler started")
	buf := new(bytes.Buffer)
	if n, err := buf.ReadFrom(r.Body); err != nil || n == 0 {
		SendErrorResponse(w, ErrorResponse{"Can't read body"}, http.StatusBadRequest)
		return
	}
	var tender Tender
	if err := json.Unmarshal(buf.Bytes(), &tender); err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Can't unmarshal"}, http.StatusBadRequest)
		return
	}
	if !CheckUsernameExists(w, tender.CreatorUsername) {
		return
	}
	if !CheckOrganizationExists(w, tender.OrganizationID.String()) {
		return
	}
	if !CheckOrganizationUser(w, tender.OrganizationID, tender.CreatorUsername) {
		return
	}
	query := `INSERT INTO tender (name, description, service_type, organization_id, creator_username)
              VALUES ($1, $2, $3, $4, $5)
              RETURNING id, status, version, created_at`
	err := db.QueryRow(context.Background(), query, tender.Name, tender.Description, tender.ServiceType, tender.OrganizationID, tender.CreatorUsername).Scan(&tender.ID, &tender.Status, &tender.Version, &tender.CreatedAt)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to create tender"}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tender)
}

func ShowTendersHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("ShowTendersHandler started")
	var err error
	url := r.URL.Query()
	query := "SELECT id, name, description, service_type, status, organization_id, version, created_at\nFROM tender\nWHERE status = 'Published'"
	args := []interface{}{}
	if st := url.Get("service_type"); st != "" {
		query += " AND service_type = $" + strconv.Itoa(len(args)+1)
		args = append(args, st)
	}
	query += "\nORDER BY name ASC"
	if lim := url.Get("limit"); lim != "" {
		limit := 5
		if limit, err = strconv.Atoi(lim); err != nil {
			SendErrorResponse(w, ErrorResponse{"Invalid limit parameter"}, http.StatusBadRequest)
			return
		}
		query += "\nLIMIT $" + strconv.Itoa(len(args)+1)
		args = append(args, limit)
	}
	if off := url.Get("offset"); off != "" {
		offset := 0
		if offset, err = strconv.Atoi(off); err != nil {
			SendErrorResponse(w, ErrorResponse{"Invalid offset parameter"}, http.StatusBadRequest)
			return
		}
		query += "\nOFFSET $" + strconv.Itoa(len(args)+1)
		args = append(args, offset)
	}
	rows, err := db.Query(context.Background(), query, args...)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find tenders"}, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var tenders []Tender
	for rows.Next() {
		var tender Tender
		err := rows.Scan(&tender.ID, &tender.Name, &tender.Description, &tender.ServiceType, &tender.Status, &tender.OrganizationID, &tender.Version, &tender.CreatedAt)
		if err != nil {
			log.Println(err.Error())
			SendErrorResponse(w, ErrorResponse{"Can't scan rows"}, http.StatusInternalServerError)
			return
		}
		tenders = append(tenders, tender)
	}
	answer, er := json.Marshal(tenders)
	if er != nil {
		log.Println(er.Error())
		SendErrorResponse(w, ErrorResponse{"Can't write answer"}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(string(answer)))
}

func ShowUsersTendersHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("ShowUsersTendersHandler started")
	var err error
	url := r.URL.Query()
	query := "SELECT id, name, description, service_type, status, organization_id, version, creator_username, created_at\nFROM tender"
	args := []interface{}{}
	if us := url.Get("username"); us != "" {
		if !CheckUsernameExists(w, us) {
			return
		}
		query += "\nWHERE creator_username = $" + strconv.Itoa(len(args)+1)
		args = append(args, us)
	} else {
		SendErrorResponse(w, ErrorResponse{"No user provided"}, http.StatusUnauthorized)
		return
	}
	query += "\nORDER BY name ASC"
	if lim := url.Get("limit"); lim != "" {
		limit := 5
		if limit, err = strconv.Atoi(lim); err != nil {
			SendErrorResponse(w, ErrorResponse{"Invalid limit parameter"}, http.StatusBadRequest)
			return
		}
		query += "\nLIMIT $" + strconv.Itoa(len(args)+1)
		args = append(args, limit)
	}
	if off := url.Get("offset"); off != "" {
		offset := 0
		if offset, err = strconv.Atoi(off); err != nil {
			SendErrorResponse(w, ErrorResponse{"Invalid offset parameter"}, http.StatusBadRequest)
			return
		}
		query += "\nOFFSET $" + strconv.Itoa(len(args)+1)
		args = append(args, offset)
	}
	rows, err := db.Query(context.Background(), query, args...)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find tenders"}, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var tenders []Tender
	for rows.Next() {
		var tender Tender
		err := rows.Scan(&tender.ID, &tender.Name, &tender.Description, &tender.ServiceType, &tender.Status, &tender.OrganizationID, &tender.Version, &tender.CreatorUsername, &tender.CreatedAt)
		if err != nil {
			log.Println(err.Error())
			SendErrorResponse(w, ErrorResponse{"Can't scan rows"}, http.StatusInternalServerError)
			return
		}
		tenders = append(tenders, tender)
	}
	answer, er := json.Marshal(tenders)
	if er != nil {
		log.Println(er.Error())
		SendErrorResponse(w, ErrorResponse{"Can't write answer"}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(string(answer)))
}

func ShowTenderStatusHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("ShowTenderStatusHandler started")
	var err error
	username := ""
	url := r.URL.Query()
	if us := url.Get("username"); us != "" {
		if !CheckUsernameExists(w, us) {
			return
		}
		username = us
	} else {
		SendErrorResponse(w, ErrorResponse{"No user provided"}, http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	tenderId := vars["tenderId"]
	if !CheckTenderExists(w, tenderId) {
		return
	}
	query := `SELECT status, organization_id
			  FROM tender t
			  WHERE t.id = $1`
	status := ""
	var org uuid.UUID
	err = db.QueryRow(context.Background(), query, tenderId).Scan(&status, &org)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find tender"}, http.StatusInternalServerError)
		return
	}
	if status != "Published" {
		if !CheckOrganizationUser(w, org, username) {
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(status))
}

func EditTenderStatusHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("EditTenderStatusHandler started")
	var err error
	username := ""
	url := r.URL.Query()
	if us := url.Get("username"); us != "" {
		if !CheckUsernameExists(w, us) {
			return
		}
		username = us
	} else {
		SendErrorResponse(w, ErrorResponse{"No user provided"}, http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	tenderId := vars["tenderId"]
	if !CheckTenderExists(w, tenderId) {
		return
	}
	var tender Tender
	if tn, ok := GetTenderInfo(w, tenderId); ok {
		tender = tn
	} else {
		return
	}
	if !CheckOrganizationUser(w, tender.OrganizationID, username) {
		return
	}
	status := ""
	if st := url.Get("status"); st != "" {
		if st == "Published" || st == "Created" || st == "Closed" {
			status = st
		} else {
			SendErrorResponse(w, ErrorResponse{"Undefined status provided"}, http.StatusBadRequest)
			return
		}
	} else {
		SendErrorResponse(w, ErrorResponse{"No status provided"}, http.StatusBadRequest)
		return
	}
	if !AddTenderToVersionsList(w, tender) {
		return
	}
	query := `UPDATE tender
			 SET status = $1, version = $2, updated_at = NOW()
			 WHERE id = $3
			 RETURNING status, version`
	err = db.QueryRow(context.Background(), query, status, tender.Version+1, tender.ID).Scan(&tender.Status, &tender.Version)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to edit tender status"}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tender)
}

func EditTenderHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("EditTenderHandler started")
	var err error
	username := ""
	url := r.URL.Query()
	if us := url.Get("username"); us != "" {
		if !CheckUsernameExists(w, us) {
			return
		}
		username = us
	} else {
		SendErrorResponse(w, ErrorResponse{"No user provided"}, http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	tenderId := vars["tenderId"]
	if !CheckTenderExists(w, tenderId) {
		return
	}
	var tender Tender
	if tn, ok := GetTenderInfo(w, tenderId); ok {
		tender = tn
	} else {
		return
	}
	if !CheckOrganizationUser(w, tender.OrganizationID, username) {
		return
	}
	if !AddTenderToVersionsList(w, tender) {
		return
	}
	buf := new(bytes.Buffer)
	if n, err := buf.ReadFrom(r.Body); err != nil || n == 0 {
		SendErrorResponse(w, ErrorResponse{"Can't read body"}, http.StatusBadRequest)
		return
	}
	if err = json.Unmarshal(buf.Bytes(), &tender); err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Can't unmarshal"}, http.StatusBadRequest)
		return
	}
	query := `UPDATE tender
			  SET name = $1, description = $2, service_type = $3, version = $4, updated_at = NOW()
			  WHERE id = $5
			  RETURNING version`
	err = db.QueryRow(context.Background(), query, tender.Name, tender.Description, tender.ServiceType, tender.Version+1, tender.ID).Scan(&tender.Version)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to edit tender"}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tender)
}

func TenderRollbackHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("TenderRollbackHandler started")
	var err error
	username := ""
	url := r.URL.Query()
	if us := url.Get("username"); us != "" {
		if !CheckUsernameExists(w, us) {
			return
		}
		username = us
	} else {
		SendErrorResponse(w, ErrorResponse{"No user provided"}, http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	tenderId := vars["tenderId"]
	vers := vars["version"]
	if !CheckTenderExists(w, tenderId) {
		return
	}
	var tender Tender
	if tn, ok := GetTenderInfo(w, tenderId); ok {
		tender = tn
	} else {
		return
	}
	if !CheckOrganizationUser(w, tender.OrganizationID, username) {
		return
	}
	if !CheckTenderVersionExists(w, tenderId, vers) {
		return
	}
	if !AddTenderToVersionsList(w, tender) {
		return
	}
	new_vers := tender.Version + 1
	if tn, ok := GetTenderVersionInfo(w, tenderId, vers); ok {
		tender = tn
	} else {
		return
	}
	query := `UPDATE tender
			  SET name = $1, description = $2, service_type = $3, version = $4, status = $5, updated_at = NOW()
			  WHERE id = $6
			  RETURNING version`
	err = db.QueryRow(context.Background(), query, tender.Name, tender.Description, tender.ServiceType, new_vers, tender.Status, tenderId).Scan(&tender.Version)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to edit tender"}, http.StatusInternalServerError)
		return
	}
	if tn, ok := GetTenderInfo(w, tenderId); ok {
		tender = tn
	} else {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tender)
}
