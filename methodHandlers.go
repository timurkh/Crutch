package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"net/http"
	"regexp"
)

const itemsPerPage = 20

type MethodHandlers struct {
	auth *AuthMiddleware
	es   *ElasticHelper
	db   *DBHelper
}

func initMethodHandlers(auth *AuthMiddleware, es *ElasticHelper, db *DBHelper) *MethodHandlers {

	mh := MethodHandlers{auth, es, db}

	return &mh
}

func stripSpecialSymbols(s string) string {
	re := regexp.MustCompile(`[*:;\.\\/()]`)
	return re.ReplaceAllString(s, " ")
}

func (mh *MethodHandlers) searchProductsHandler(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.auth.getUserInfo(r)

	cities, err := mh.db.getUserConsigneeCities(r.Context(), userInfo.Id)

	var searchQuery SearchQuery
	err = json.NewDecoder(r.Body).Decode(&searchQuery)
	if err != nil {
		log.Warn("Failed to decode search params: ", err)
		return err
	}

	log.Printf("## Handling search request text=%s, category=%s, code=%s, name=%s, property=%s, page=%v\n", searchQuery.Text, searchQuery.Category, searchQuery.Code, searchQuery.Name, searchQuery.Property, searchQuery.Page)
	totalPages := 0
	entries := make([]map[string]string, 0)
	for {
		hits, totalPages_, err := mh.es.search(&searchQuery, r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		entries_, err := mh.getResponseEntries(r.Context(), hits, userInfo.Id, searchQuery.CityID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		entries = append(entries, entries_...)
		totalPages = totalPages_

		if len(entries) > itemsPerPage/2 || searchQuery.Page >= totalPages-1 {
			break
		}
		searchQuery.Page++
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		UserInfo
		Cities     []City      `json:"cities"`
		Page       int         `json:"page"`
		TotalPages int         `json:"totalPages"`
		Results    interface{} `json:"results"`
	}{userInfo, cities, searchQuery.Page, totalPages, entries})

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil

}

func (mh *MethodHandlers) getResponseEntries(ctx context.Context, hits []interface{}, userId int, cityId int) ([]map[string]string, error) {

	ids := make([]int, len(hits))

	// iterate through hits
	for i, hit := range hits {
		h := hit.(map[string]interface{})
		id, _ := strconv.Atoi(h["_id"].(string))
		ids[i] = id
	}

	log.Debug("Quering details for product_ids ", ids)

	products, err := mh.db.getProductEntries(ctx, ids, userId, cityId)

	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve list of products: %v", err)
	}

	return products, err
}

func (mh *MethodHandlers) getResponseEntriesFromElastic(hits []interface{}) []map[string]string {

	entries := make([]map[string]string, len(hits))

	// iterate through hits
	for i, hit := range hits {
		h := hit.(map[string]interface{})
		s := h["_source"].(map[string]interface{})
		var categories string

		//categories
		for i, c := range s["category"].([]interface{}) {
			if i > 0 {
				categories += ", "
			}
			cat := c.(map[string]interface{})["name"]
			if cat != nil {
				categories += cat.(string)
			}
		}

		//properties
		var properties string
		if s["properties"] != nil {
			for i, p := range s["properties"].([]interface{}) {
				if i > 0 {
					properties += ", "
				}
				properties += p.(map[string]interface{})["property"].(map[string]interface{})["name"].(string)
				properties += " : "
				properties += p.(map[string]interface{})["value"].(string)
			}
		}

		entry := map[string]string{
			"id":         h["_id"].(string),
			"categories": categories,
			"code":       s["code"].(string),
			"name":       s["name"].(string),
			"properties": properties,
		}

		entries[i] = entry
	}

	return entries

}

func (mh *MethodHandlers) getCurrentUser(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.auth.getUserInfo(r)

	cities, err := mh.db.getUserConsigneeCities(r.Context(), userInfo.Id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		UserInfo
		Cities []City `json:"cities"`
	}{userInfo, cities})

	log.Info(userInfo, cities)

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil

}
