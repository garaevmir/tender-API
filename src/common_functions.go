package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
)

type ErrorResponse struct {
	Reason string `json:"reason"`
}

func SendErrorResponse(w http.ResponseWriter, text ErrorResponse, err int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err)
	json.NewEncoder(w).Encode(text)
}

func CheckOrganizationUser(w http.ResponseWriter, org uuid.UUID, username string) bool {
	var exists bool
	query := `SELECT EXISTS (
			  SELECT 1
			  FROM organization_responsible ore
			  JOIN employee e ON ore.user_id = e.id
			  WHERE ore.organization_id = $1
			  AND e.username = $2);`
	err := db.QueryRow(context.Background(), query, org, username).Scan(&exists)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to authorize"}, http.StatusInternalServerError)
		return false
	}
	if !exists {
		SendErrorResponse(w, ErrorResponse{"Don't have rights"}, http.StatusForbidden)
		return false
	}
	return true
}

func CheckUsernameExists(w http.ResponseWriter, username string) bool {
	var exists bool
	query := `SELECT EXISTS (
			  SELECT 1
			  FROM employee
			  WHERE username = $1);`
	err := db.QueryRow(context.Background(), query, username).Scan(&exists)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find user"}, http.StatusInternalServerError)
		return false
	}
	if !exists {
		SendErrorResponse(w, ErrorResponse{"No such user"}, http.StatusUnauthorized)
		return false
	}
	return true
}

func CheckTenderExists(w http.ResponseWriter, tenderId string) bool {
	var exists bool
	query := `SELECT EXISTS (
			  SELECT 1
			  FROM tender
			  WHERE id = $1);`
	err := db.QueryRow(context.Background(), query, tenderId).Scan(&exists)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find tender"}, http.StatusInternalServerError)
		return false
	}
	if !exists {
		SendErrorResponse(w, ErrorResponse{"No such tender"}, http.StatusNotFound)
		return false
	}
	return true
}

func AddTenderToVersionsList(w http.ResponseWriter, tender Tender) bool {
	var id uuid.UUID
	query := `INSERT INTO tender_version (tender_id, version, name, description, service_type, status)
              VALUES ($1, $2, $3, $4, $5, $6)
              RETURNING id`
	err := db.QueryRow(context.Background(), query, tender.ID, tender.Version, tender.Name, tender.Description, tender.ServiceType, tender.Status).Scan(&id)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to create reserve copy of tender"}, http.StatusInternalServerError)
		return false
	}
	return true
}

func GetTenderInfo(w http.ResponseWriter, tenderId string) (Tender, bool) {
	var tender Tender
	query := `SELECT id, name, description, service_type, status, organization_id, creator_username, version, created_at
			  FROM tender t
			  WHERE t.id = $1`
	err := db.QueryRow(context.Background(), query, tenderId).Scan(&tender.ID, &tender.Name, &tender.Description, &tender.ServiceType, &tender.Status, &tender.OrganizationID, &tender.CreatorUsername, &tender.Version, &tender.CreatedAt)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find tender"}, http.StatusInternalServerError)
		return tender, false
	}
	return tender, true
}

func CheckTenderVersionExists(w http.ResponseWriter, tenderId, vers string) bool {
	var exists bool
	query := `SELECT EXISTS (
			  SELECT 1
			  FROM tender_version
			  WHERE tender_id = $1 AND version = $2);`
	err := db.QueryRow(context.Background(), query, tenderId, vers).Scan(&exists)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find tender version"}, http.StatusInternalServerError)
		return false
	}
	if !exists {
		SendErrorResponse(w, ErrorResponse{"No such tender"}, http.StatusNotFound)
		return false
	}
	return true
}

func GetTenderVersionInfo(w http.ResponseWriter, tenderId, vers string) (Tender, bool) {
	var tender Tender
	query := `SELECT name, description, service_type, status
			  FROM tender_version t
			  WHERE t.tender_id = $1 AND t.version = $2`
	err := db.QueryRow(context.Background(), query, tenderId, vers).Scan(&tender.Name, &tender.Description, &tender.ServiceType, &tender.Status)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find tender version info"}, http.StatusInternalServerError)
		return tender, false
	}
	return tender, true
}

func CheckOrganizationExists(w http.ResponseWriter, id string) bool {
	var exists bool
	query := `SELECT EXISTS (
			  SELECT 1
			  FROM organization
			  WHERE id = $1);`
	err := db.QueryRow(context.Background(), query, id).Scan(&exists)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find organization"}, http.StatusInternalServerError)
		return false
	}
	if !exists {
		SendErrorResponse(w, ErrorResponse{"No such organization"}, http.StatusNotFound)
		return false
	}
	return true
}

func GetUserId(w http.ResponseWriter, username string) (string, bool) {
	user_id := ""
	query := `SELECT id
			  FROM employee e
			  WHERE e.username = $1`
	err := db.QueryRow(context.Background(), query, username).Scan(&user_id)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find user"}, http.StatusInternalServerError)
		return user_id, false
	}
	return user_id, true
}

func GetOrganizationId(w http.ResponseWriter, author_id string) (uuid.UUID, bool) {
	var organization_id uuid.UUID
	query := `SELECT organization_id
			  FROM organization_responsible
			  WHERE user_id = $1`
	err := db.QueryRow(context.Background(), query, author_id).Scan(&organization_id)
	if err != nil {
		log.Println(err.Error())
		log.Println("Failed to find organization")
		return organization_id, false
	}
	return organization_id, true
}

func CheckBidExists(w http.ResponseWriter, bidId string) bool {
	var exists bool
	query := `SELECT EXISTS (
			  SELECT 1
			  FROM bid
			  WHERE id = $1);`
	err := db.QueryRow(context.Background(), query, bidId).Scan(&exists)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find bid"}, http.StatusInternalServerError)
		return false
	}
	if !exists {
		SendErrorResponse(w, ErrorResponse{"No such bid"}, http.StatusNotFound)
		return false
	}
	return true
}

func GetBidInfo(w http.ResponseWriter, bidId string) (Bid, bool) {
	var bid Bid
	query := `SELECT id, name, description, status, tender_id, author_type, author_id, organization_id, version, decision, approved_count, created_at
			  FROM bid b
			  WHERE b.id = $1`
	err := db.QueryRow(context.Background(), query, bidId).Scan(&bid.ID, &bid.Name, &bid.Description, &bid.Status, &bid.TenderID, &bid.AuthorType, &bid.AuthorID, &bid.OrganizationID, &bid.Version, &bid.Decision, &bid.ApprovedCount, &bid.CreatedAt)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find bid"}, http.StatusInternalServerError)
		return bid, false
	}
	return bid, true
}

func AddBidToVersionsList(w http.ResponseWriter, bid Bid) bool {
	var id uuid.UUID
	query := `INSERT INTO bid_version (bid_id, version, name, description, decision, approved_count, status)
              VALUES ($1, $2, $3, $4, $5, $6, $7)
              RETURNING id`
	err := db.QueryRow(context.Background(), query, bid.ID, bid.Version, bid.Name, bid.Description, bid.Decision, bid.ApprovedCount, bid.Status).Scan(&id)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to create reserve copy of bid"}, http.StatusInternalServerError)
		return false
	}
	return true
}

func ClosingTender(w http.ResponseWriter, tenderId string) bool {
	var tender Tender
	if tn, ok := GetTenderInfo(w, tenderId); ok {
		tender = tn
	} else {
		return false
	}
	query := `UPDATE tender
			  SET status = 'Closed', version = $1, updated_at = NOW()
			  WHERE id = $2
			  RETURNING status, version`
	err := db.QueryRow(context.Background(), query, tender.Version+1, tender.ID).Scan(&tender.Status, &tender.Version)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to edit tender status"}, http.StatusInternalServerError)
		return false
	}
	return true
}

func MakeDecision(w http.ResponseWriter, bid *Bid, decision string) bool {
	query := `UPDATE bid
			  SET decision = $1, approved_count = $2, status = 'Canceled', version = $3, updated_at = NOW()
			  WHERE id = $4
			  RETURNING decision, approved_count, version`
	err := db.QueryRow(context.Background(), query, decision, bid.ApprovedCount+1, bid.Version+1, bid.ID).Scan(&bid.Decision, &bid.ApprovedCount, &bid.Version)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to edit bid decision"}, http.StatusInternalServerError)
		return false
	}
	return true
}

func CheckUserApproveExists(w http.ResponseWriter, bid Bid, username string) bool {
	var exists bool
	query := `SELECT EXISTS (
			  SELECT 1
			  FROM bid_approve
			  WHERE bid_id = $1 AND username = $2);`
	err := db.QueryRow(context.Background(), query, bid.ID, username).Scan(&exists)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find organization"}, http.StatusInternalServerError)
		return false
	}
	if exists {
		SendErrorResponse(w, ErrorResponse{"Bid already approved by user"}, http.StatusNotFound)
		return false
	}
	return true
}

func AddApproveBid(w http.ResponseWriter, bid *Bid) bool {
	query := `UPDATE bid
			  SET approved_count = $1, version = $2, updated_at = NOW()
			  WHERE id = $3
			  RETURNING approved_count, version`
	err := db.QueryRow(context.Background(), query, bid.ApprovedCount+1, bid.Version+1, bid.ID).Scan(&bid.ApprovedCount, &bid.Version)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to edit bid decision"}, http.StatusInternalServerError)
		return false
	}
	return true
}

func CountResponsibles(w http.ResponseWriter, bid Bid) (int, bool) {
	resp := 0
	query := `SELECT count(user_id)
			  FROM organization_responsible or2 
			  WHERE organization_id = $1`
	err := db.QueryRow(context.Background(), query, bid.OrganizationID).Scan(&resp)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to count responsible"}, http.StatusInternalServerError)
		return resp, false
	}
	return resp, true
}

func CheckBidVersionExists(w http.ResponseWriter, bidId, vers string) bool {
	var exists bool
	query := `SELECT EXISTS (
			  SELECT 1
			  FROM bid_version
			  WHERE bid_id = $1 AND version = $2);`
	err := db.QueryRow(context.Background(), query, bidId, vers).Scan(&exists)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find bid version"}, http.StatusInternalServerError)
		return false
	}
	if !exists {
		SendErrorResponse(w, ErrorResponse{"No such bid"}, http.StatusNotFound)
		return false
	}
	return true
}

func GetBidVersionInfo(w http.ResponseWriter, bidId, vers string) (Bid, bool) {
	var bid Bid
	query := `SELECT name, description, status
			  FROM bid_version 
			  WHERE bid_id = $1 AND version = $2`
	err := db.QueryRow(context.Background(), query, bidId, vers).Scan(&bid.Name, &bid.Description, &bid.Status)
	if err != nil {
		log.Println(err.Error())
		SendErrorResponse(w, ErrorResponse{"Failed to find bid version info"}, http.StatusInternalServerError)
		return bid, false
	}
	return bid, true
}
