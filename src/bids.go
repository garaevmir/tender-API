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

type BidReview struct {
	ID          uuid.UUID `json:"id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Bid struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Status         string    `json:"status"`
	TenderID       uuid.UUID `json:"tenderId"`
	AuthorType     string    `json:"authorType"`
	AuthorID       uuid.UUID `json:"authorId"`
	OrganizationID uuid.UUID `json:"-"`
	Decision       string    `json:"-"`
	ApprovedCount  int       `json:"-"`
	Version        int       `json:"version"`
	CreatedAt      time.Time `json:"createdAt"`
}

func CreateBidHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("CreateBidHandler started")
	buf := new(bytes.Buffer)
	if n, err := buf.ReadFrom(r.Body); err != nil || n == 0 {
		SendErrorResponse(w, ErrorResponse{"Can't read body"}, http.StatusBadRequest)
		return
	}
	var bid Bid
	if err := json.Unmarshal(buf.Bytes(), &bid); err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Can't unmarshal"}, http.StatusBadRequest)
		return
	}
	if bid.AuthorType == "Organization" {
		if !CheckOrganizationExists(w, bid.AuthorID.String()) {
			return
		}
		bid.OrganizationID = bid.AuthorID
	} else {
		if oi, ok := GetOrganizationId(w, bid.AuthorID.String()); ok {
			bid.OrganizationID = oi
		} else {
			return
		}
	}
	if !CheckTenderExists(w, bid.TenderID.String()) {
		return
	}
	var tender Tender
	if tn, ok := GetTenderInfo(w, bid.TenderID.String()); ok {
		tender = tn
	} else {
		return
	}
	if tender.OrganizationID != bid.OrganizationID && tender.Status != "Published" {
		SendErrorResponse(w, ErrorResponse{"Don't have rights"}, http.StatusForbidden)
		return
	}
	query := `INSERT INTO bid (name, description, tender_id, author_type, author_id, organization_id)
              VALUES ($1, $2, $3, $4, $5, $6)
              RETURNING id, status, version, decision, created_at`
	err := db.QueryRow(context.Background(), query, bid.Name, bid.Description, bid.TenderID, bid.AuthorType, bid.AuthorID, tender.OrganizationID).Scan(&bid.ID, &bid.Status, &bid.Version, &bid.Decision, &bid.CreatedAt)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to create tender"}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bid)
}

func ShowUsersBidsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("ShowUsersBidsHandler started")
	var err error
	url := r.URL.Query()
	query := "SELECT id, name, description, status, tender_id, author_type, author_id, organization_id, version, decision, approved_count, created_at\nFROM bid"
	args := []interface{}{}
	if us := url.Get("username"); us != "" {
		user_id := ""
		if ui, ok := GetUserId(w, us); ok {
			user_id = ui
		} else {
			return
		}
		query += "\nWHERE author_id = $" + strconv.Itoa(len(args)+1)
		args = append(args, user_id)
	} else {
		SendErrorResponse(w, ErrorResponse{"No user provided"}, http.StatusUnauthorized)
		return
	}
	query += "\nORDER BY name ASC"
	if lim := url.Get("limit"); lim != "" {
		limit := 5
		if limit, err = strconv.Atoi(lim); err != nil {
			log.Println(err.Error())
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
		SendErrorResponse(w, ErrorResponse{"Failed to find bids"}, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var bids []Bid
	for rows.Next() {
		var bid Bid
		err := rows.Scan(&bid.ID, &bid.Name, &bid.Description, &bid.Status, &bid.TenderID, &bid.AuthorType, &bid.AuthorID, &bid.OrganizationID, &bid.Version, &bid.Decision, &bid.ApprovedCount, &bid.CreatedAt)
		if err != nil {
			log.Println(err.Error())
			SendErrorResponse(w, ErrorResponse{"Can't scan rows"}, http.StatusInternalServerError)
			return
		}
		bids = append(bids, bid)
	}
	answer, er := json.Marshal(bids)
	if er != nil {
		log.Println(er.Error())
		SendErrorResponse(w, ErrorResponse{"Can't write answer"}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(string(answer)))
}

func ShowTenderBidsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("ShowTenderBidsHandler started")
	var err error
	username := ""
	url := r.URL.Query()
	vars := mux.Vars(r)
	tenderId := vars["tenderId"]
	query := "SELECT id, name, description, status, tender_id, author_type, author_id, organization_id, version, decision, approved_count, created_at\nFROM bid\n"
	args := []interface{}{}
	if us := url.Get("username"); us != "" {
		username = us
		user_id := ""
		if ui, ok := GetUserId(w, us); ok {
			user_id = ui
		} else {
			return
		}
		var organization_id uuid.UUID
		if oi, ok := GetOrganizationId(w, user_id); ok {
			organization_id = oi
		}
		query += "\nWHERE (status = 'Published' OR organization_id = $1) AND tender_id = $2"
		args = append(args, organization_id, tenderId)
	} else {
		SendErrorResponse(w, ErrorResponse{"No user provided"}, http.StatusUnauthorized)
		return
	}
	if !CheckTenderExists(w, tenderId) {
		return
	}
	var tender Tender
	if tn, ok := GetTenderInfo(w, tenderId); ok {
		tender = tn
	} else {
		return
	}
	if tender.Status != "Published" {
		if !CheckOrganizationUser(w, tender.OrganizationID, username) {
			return
		}
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
			log.Println(err.Error())
			SendErrorResponse(w, ErrorResponse{"Invalid offset parameter"}, http.StatusBadRequest)
			return
		}
		query += "\nOFFSET $" + strconv.Itoa(len(args)+1)
		args = append(args, offset)
	}
	rows, err := db.Query(context.Background(), query, args...)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find bids"}, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var bids []Bid
	for rows.Next() {
		var bid Bid
		err := rows.Scan(&bid.ID, &bid.Name, &bid.Description, &bid.Status, &bid.TenderID, &bid.AuthorType, &bid.AuthorID, &bid.OrganizationID, &bid.Version, &bid.Decision, &bid.ApprovedCount, &bid.CreatedAt)
		if err != nil {
			log.Println(err.Error())
			SendErrorResponse(w, ErrorResponse{"Can't scan rows"}, http.StatusInternalServerError)
			return
		}
		bids = append(bids, bid)
	}
	answer, er := json.Marshal(bids)
	if er != nil {
		log.Println(er.Error())
		SendErrorResponse(w, ErrorResponse{"Can't write answer"}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(string(answer)))
}

func ShowBidStatusHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("ShowBidStatusHandler started")
	username := ""
	vars := mux.Vars(r)
	bidId := vars["bidId"]
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
	if !CheckBidExists(w, bidId) {
		return
	}
	var bid Bid
	if bd, ok := GetBidInfo(w, bidId); ok {
		bid = bd
	} else {
		return
	}
	if bid.Status != "Published" {
		if !CheckOrganizationUser(w, bid.OrganizationID, username) {
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(bid.Status))
}

func EditBidStatusHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("EditBidStatusHandler started")
	username := ""
	vars := mux.Vars(r)
	bidId := vars["bidId"]
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
	if !CheckBidExists(w, bidId) {
		return
	}
	var bid Bid
	if bd, ok := GetBidInfo(w, bidId); ok {
		bid = bd
	} else {
		return
	}
	if !CheckOrganizationUser(w, bid.OrganizationID, username) {
		return
	}
	if !AddBidToVersionsList(w, bid) {
		return
	}
	status := ""
	if st := url.Get("status"); st != "" {
		status = st
	}
	query := `UPDATE bid
			  SET status = $1, version = $2, updated_at = NOW()
			  WHERE id = $3
			  RETURNING status, version`
	err := db.QueryRow(context.Background(), query, status, bid.Version+1, bid.ID).Scan(&bid.Status, &bid.Version)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to edit bid status"}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bid)
}

func EditBidHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("EditBidHandler started")
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
	bidId := vars["bidId"]
	if !CheckBidExists(w, bidId) {
		return
	}
	var bid Bid
	if bd, ok := GetBidInfo(w, bidId); ok {
		bid = bd
	} else {
		return
	}
	if !CheckOrganizationUser(w, bid.OrganizationID, username) {
		return
	}
	if !AddBidToVersionsList(w, bid) {
		return
	}
	buf := new(bytes.Buffer)
	if n, err := buf.ReadFrom(r.Body); err != nil || n == 0 {
		SendErrorResponse(w, ErrorResponse{"Can't read body"}, http.StatusBadRequest)
		return
	}
	if err = json.Unmarshal(buf.Bytes(), &bid); err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Can't unmarshal"}, http.StatusBadRequest)
		return
	}
	query := `UPDATE bid
			  SET name = $1, description = $2, version = $3, updated_at = NOW()
			  WHERE id = $4
			  RETURNING version`
	err = db.QueryRow(context.Background(), query, bid.Name, bid.Description, bid.Version+1, bid.ID).Scan(&bid.Version)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to edit bid"}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bid)
}

func SubmitDecisionHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("SubmitDecisionHandler started")
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
	bidId := vars["bidId"]
	if !CheckBidExists(w, bidId) {
		return
	}
	var bid Bid
	if bd, ok := GetBidInfo(w, bidId); ok {
		bid = bd
	} else {
		return
	}
	if !CheckTenderExists(w, bid.TenderID.String()) {
		return
	}
	var tender Tender
	if tn, ok := GetTenderInfo(w, bid.TenderID.String()); ok {
		tender = tn
	} else {
		return
	}
	if !CheckOrganizationUser(w, tender.OrganizationID, username) {
		return
	}
	if bid.Decision != "None" {
		SendErrorResponse(w, ErrorResponse{"Decision already made"}, http.StatusBadRequest)
		return
	}
	decision := ""
	if dc := url.Get("decision"); dc != "" {
		if dc != "Approved" && dc != "Rejected" {
			SendErrorResponse(w, ErrorResponse{"Undefined decision"}, http.StatusBadRequest)
			return
		}
		decision = dc
	} else {
		SendErrorResponse(w, ErrorResponse{"No decision provided"}, http.StatusNotFound)
		return
	}
	if !AddBidToVersionsList(w, bid) {
		return
	}
	if decision == "Rejected" {
		bid.ApprovedCount--
		if !MakeDecision(w, &bid, decision) {
			return
		}
		if !ClosingTender(w, bid.TenderID.String()) {
			return
		}
	} else {
		resp := 0
		if rs, ok := CountResponsibles(w, bid); ok {
			resp = rs
		} else {
			return
		}
		if !CheckUserApproveExists(w, bid, username) {
			return
		}
		id := ""
		query := `INSERT INTO bid_approve (bid_id, username)
				  VALUES ($1, $2)
				  RETURNING id`
		err := db.QueryRow(context.Background(), query, bid.ID, username).Scan(&id)
		if err != nil {
			log.Println(err.Error())
			SendErrorResponse(w, ErrorResponse{"Failed to submit approvement"}, http.StatusInternalServerError)
			return
		}
		if bid.ApprovedCount+1 >= min(3, resp) {
			if !MakeDecision(w, &bid, decision) {
				return
			}
			if !ClosingTender(w, bid.TenderID.String()) {
				return
			}
		} else {
			if !AddApproveBid(w, &bid) {
				return
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bid)
}

func BidRollbackHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("BidRollbackHandler started")
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
	bidId := vars["bidId"]
	vers := vars["version"]
	if !CheckBidExists(w, bidId) {
		return
	}
	var bid Bid
	if tn, ok := GetBidInfo(w, bidId); ok {
		bid = tn
	} else {
		return
	}
	if !CheckOrganizationUser(w, bid.OrganizationID, username) {
		return
	}
	if !CheckBidVersionExists(w, bidId, vers) {
		return
	}
	if !AddBidToVersionsList(w, bid) {
		return
	}
	new_vers := bid.Version + 1
	if tn, ok := GetBidVersionInfo(w, bidId, vers); ok {
		bid = tn
	} else {
		return
	}
	query := `UPDATE bid
			  SET name = $1, description = $2, version = $3, status = $4, updated_at = NOW()
			  WHERE id = $5
			  RETURNING version`
	err = db.QueryRow(context.Background(), query, bid.Name, bid.Description, new_vers, bid.Status, bidId).Scan(&bid.Version)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to edit bid"}, http.StatusInternalServerError)
		return
	}
	if tn, ok := GetBidInfo(w, bidId); ok {
		bid = tn
	} else {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bid)
}

func BidReviewHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("BidReviewHandler started")
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
	bidId := vars["bidId"]
	if !CheckBidExists(w, bidId) {
		return
	}
	var bid Bid
	if bd, ok := GetBidInfo(w, bidId); ok {
		bid = bd
	} else {
		return
	}
	if !CheckOrganizationUser(w, bid.OrganizationID, username) {
		return
	}
	review := ""
	if rv := url.Get("bidFeedback"); rv != "" {
		review = rv
	} else {
		SendErrorResponse(w, ErrorResponse{"No review provided"}, http.StatusBadRequest)
		return
	}
	id := ""
	query := `INSERT INTO bid_review (bid_id, username, review)
              VALUES ($1, $2, $3)
              RETURNING id`
	err := db.QueryRow(context.Background(), query, bidId, username, review).Scan(&id)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to create review"}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bid)
}

func ShowBidReviewsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("ShowBidReviewsHandler started")
	var err error
	vars := mux.Vars(r)
	tenderId := vars["tenderId"]
	author_id := ""
	url := r.URL.Query()
	if us := url.Get("authorUsername"); us != "" {
		if !CheckUsernameExists(w, us) {
			return
		}
		if ai, ok := GetUserId(w, us); ok {
			author_id = ai
		} else {
			return
		}
	} else {
		SendErrorResponse(w, ErrorResponse{"No author provided"}, http.StatusUnauthorized)
		return
	}
	requestor_username := ""
	if us := url.Get("requesterUsername"); us != "" {
		if !CheckUsernameExists(w, us) {
			return
		}
		requestor_username = us
	} else {
		SendErrorResponse(w, ErrorResponse{"No requestor provided"}, http.StatusUnauthorized)
		return
	}
	if !CheckTenderExists(w, tenderId) {
		return
	}
	var tender Tender
	if tn, ok := GetTenderInfo(w, tenderId); ok {
		tender = tn
	} else {
		return
	}
	if !CheckOrganizationUser(w, tender.OrganizationID, requestor_username) {
		return
	}
	args := []interface{}{}
	args = append(args, author_id, tenderId)
	query := `SELECT br.id, br.review, br.created_at 
			  FROM bid b
			  JOIN bid_review br 
			  ON b.id = br.bid_id
			  WHERE author_id = $1 and author_type = 'User' and tender_id != $2`
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
		SendErrorResponse(w, ErrorResponse{"Failed to find reviews"}, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var bids []BidReview
	for rows.Next() {
		var br BidReview
		err := rows.Scan(&br.ID, &br.Description, &br.CreatedAt)
		if err != nil {
			log.Println(err.Error())
			SendErrorResponse(w, ErrorResponse{"Can't scan rows"}, http.StatusInternalServerError)
			return
		}
		bids = append(bids, br)
	}
	answer, er := json.Marshal(bids)
	if er != nil {
		log.Println(er.Error())
		SendErrorResponse(w, ErrorResponse{"Can't write answer"}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(string(answer)))
}
