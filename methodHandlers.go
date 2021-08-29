package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/xuri/excelize/v2"
)

const itemsPerPage = 100

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

func (mh *MethodHandlers) searchProductsHandler(w http.ResponseWriter, r *http.Request) (err error) {

	userInfo := mh.auth.getUserInfo(r)

	var cities []City

	if !userInfo.Admin {
		cities, err = mh.db.getUserConsigneeCities(r.Context(), userInfo)
		if err != nil {
			err = fmt.Errorf("Failed to get consignee cities: %s", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}

		if len(cities) == 0 {
			err = fmt.Errorf("Current user does not have any warehouses assigned")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}
	}

	var searchQuery SearchQuery
	err = schema.NewDecoder().Decode(&searchQuery, r.URL.Query())
	if err != nil {
		err = fmt.Errorf("Failed to decode search params: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	log.Printf("## Handling search request text=%s, category=%s, code=%s, name=%s, property=%s, page=%v\n", searchQuery.Text, searchQuery.Category, searchQuery.Code, searchQuery.Name, searchQuery.Property, searchQuery.Page)
	totalPages := 0
	entries := make([]map[string]interface{}, 0)
	for {
		hits, totalPages_, err := mh.es.search(&searchQuery, r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		entries_, err := mh.getResponseEntries(r.Context(), hits, userInfo, searchQuery.CityID)
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

func (mh *MethodHandlers) getResponseEntries(ctx context.Context, hits []interface{}, userInfo UserInfo, cityId int) ([]map[string]interface{}, error) {

	ids := make([]int, len(hits))

	// iterate through hits
	products_score := make(map[int]float64, 0)
	for i, hit := range hits {
		h := hit.(map[string]interface{})
		id, _ := strconv.Atoi(h["_id"].(string))
		ids[i] = id
		score, _ := h["_score"].(float64)
		products_score[id] = score
	}

	log.Debug("Quering details for product_ids ", products_score)

	products, err := mh.db.getProductEntries(ctx, ids, products_score, userInfo, cityId)

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

	cities, err := mh.db.getUserConsigneeCities(r.Context(), userInfo)

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

func (mh *MethodHandlers) getOrdersHandler(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.auth.getUserInfo(r)

	orders, err := mh.db.getOrders(r.Context(), userInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		Orders interface{} `json:"orders"`
	}{orders})

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}

func (mh *MethodHandlers) getOrderHandler(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.auth.getUserInfo(r)

	params := mux.Vars(r)
	orderId_ := params["orderId"]
	orderId, err := strconv.Atoi(orderId_)
	if err != nil {
		err = fmt.Errorf("Failed to determin requested order ID: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	orderDetails, err := mh.db.getOrder(r.Context(), userInfo, orderId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(orderDetails)

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}

func (mh *MethodHandlers) getOrdersExcelHandler(w http.ResponseWriter, r *http.Request) error {

	file, err := os.CreateTemp("/tmp", "*.xlsx")
	if err != nil {
		log.Error(err)
		return err
	}
	defer os.Remove(file.Name())

	xls := excelize.NewFile()

	xls.SetCellValue("Sheet1", "A1", "[8]")
	xls.SetCellValue("Sheet1", "B1", "[8]")
	xls.SetCellValue("Sheet1", "D1", "(2)")
	xls.SetCellValue("Sheet1", "E1", "(2а)")
	xls.SetCellValue("Sheet1", "F1", "(2б)")
	xls.SetCellValue("Sheet1", "G1", "(3)")
	xls.SetCellValue("Sheet1", "H1", "(3)")
	xls.SetCellValue("Sheet1", "I1", "(4)")
	xls.SetCellValue("Sheet1", "J1", "(4)")
	xls.SetCellValue("Sheet1", "K1", "(6а)")
	xls.SetCellValue("Sheet1", "O1", "(6б)")
	xls.SetCellValue("Sheet1", "Q1", "Табличная часть УПД")
	xls.SetCellValue("Sheet1", "S1", "Табличная часть УПД")
	xls.SetCellValue("Sheet1", "T1", "Табличная часть УПД")
	xls.SetCellValue("Sheet1", "U1", "Табличная часть УПД")
	xls.SetCellValue("Sheet1", "V1", "Табличная часть УПД")
	xls.SetCellValue("Sheet1", "W1", "Табличная часть УПД")
	xls.SetCellValue("Sheet1", "X1", "Табличная часть УПД")
	xls.SetCellValue("Sheet1", "Y1", "Табличная часть УПД")
	xls.SetCellValue("Sheet1", "Z1", "[11]")
	xls.SetCellValue("Sheet1", "A2", "Номер заказа")
	xls.SetCellValue("Sheet1", "B2", "Дата заказа")
	xls.SetCellValue("Sheet1", "C2", "Дата отгрузки")
	xls.SetCellValue("Sheet1", "D2", "Продавец")
	xls.SetCellValue("Sheet1", "E2", "Адрес")
	xls.SetCellValue("Sheet1", "F2", "ИНН/КПП продавца")
	xls.SetCellValue("Sheet1", "G2", "Грузоотправитель")
	xls.SetCellValue("Sheet1", "H2", "Адрес прузоотправителя")
	xls.SetCellValue("Sheet1", "I2", "Грузополучатель")
	xls.SetCellValue("Sheet1", "J2", "Адрес прузополучателя")
	xls.SetCellValue("Sheet1", "K2", "Покупатель")
	xls.SetCellValue("Sheet1", "L2", "Адрес покупателя")
	xls.SetCellValue("Sheet1", "M2", "ИНН поставщика")
	xls.SetCellValue("Sheet1", "N2", "КПП поставщика")
	xls.SetCellValue("Sheet1", "O2", "ИНН покупателя")
	xls.SetCellValue("Sheet1", "P2", "КПП покупателя")
	xls.SetCellValue("Sheet1", "Q2", "Код товара/работ, услуг")
	xls.SetCellValue("Sheet1", "R2", "Наименование товара")
	xls.SetCellValue("Sheet1", "S2", "Единица обозначения - условное обозначение (национальное)")
	xls.SetCellValue("Sheet1", "T2", "Количество (объём)")
	xls.SetCellValue("Sheet1", "U2", "Цена (тариф) за единицу измерения")
	xls.SetCellValue("Sheet1", "V2", "Стоимость товаров (работ, услуг), имущественных прав без налога - всего")
	xls.SetCellValue("Sheet1", "W2", "Налоговая ставка")
	xls.SetCellValue("Sheet1", "X2", "Сумма налога, предъявляемая покупателю")
	xls.SetCellValue("Sheet1", "Y2", "Стоимость товаров (работ, услуг), имущественных прав с налогом - всего")
	xls.SetCellValue("Sheet1", "Z2", "Дата отгрузки, передачи (сдачи)")

	userInfo := mh.auth.getUserInfo(r)
	orders, err := mh.db.getOrders(r.Context(), userInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	row := 3
	for _, order := range orders {
		xls.SetCellValue("Sheet1", fmt.Sprintf("A%v", row), order["contractor_number"])
		row++
	}

	xls.SaveAs(file.Name())

	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote("Заказы.xlsx"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, file.Name())

	return nil
}
