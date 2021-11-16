package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode/utf8"

	"net/http"
	"regexp"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/xuri/excelize/v2"

	gorilla_context "github.com/gorilla/context"
)

const itemsPerPage = 200

type MethodHandlers struct {
	es       *ElasticHelper
	prodDB   *ProdDBHelper
	crutchDB *CrutchDBHelper
}

func initMethodHandlers(es *ElasticHelper, db *ProdDBHelper, crutchDb *CrutchDBHelper) *MethodHandlers {

	mh := MethodHandlers{es, db, crutchDb}

	return &mh
}

func stripSpecialSymbols(s string) string {
	re := regexp.MustCompile(`[*:;\.\\/()]`)
	return re.ReplaceAllString(s, " ")
}

type SearchResults struct {
	UserInfo
	Cities     []City              `json:"cities"`
	Page       int                 `json:"page"`
	TotalPages int                 `json:"totalPages"`
	Results    []SearchResultEntry `json:"results"`
}

func (mh *MethodHandlers) getUserInfo(r *http.Request) UserInfo {
	return gorilla_context.Get(r, "UserInfo").(UserInfo)
}

func (mh *MethodHandlers) searchProductsHandler(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.getUserInfo(r)

	var searchQuery SearchQuery
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err := decoder.Decode(&searchQuery, r.URL.Query())
	if err != nil {
		err = fmt.Errorf("Failed to decode search params: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	sr, err, status := mh.searchProducts(r.Context(), userInfo, searchQuery)
	if err != nil {
		http.Error(w, err.Error(), status)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	err = json.NewEncoder(w).Encode(sr)
	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}

func (mh *MethodHandlers) searchProducts(ctx context.Context, userInfo UserInfo, searchQuery SearchQuery) (sr *SearchResults, err error, status int) {

	log.Info(fmt.Printf("Handling search request text=%s, category=%s, code=%s, name=%s, property=%s, page=%v\n", searchQuery.Text, searchQuery.Category, searchQuery.Code, searchQuery.Name, searchQuery.Property, searchQuery.Page))

	var cities []City

	if !userInfo.Admin {
		cities, err = mh.prodDB.getUserConsigneeCities(ctx, userInfo)
		if err != nil {
			err = fmt.Errorf("Failed to get consignee cities: %s", err.Error())
			return nil, err, http.StatusBadRequest
		}

		if len(cities) == 0 && userInfo.SupplierId != 0 {
			cities, err = mh.prodDB.getSupplierCities(ctx, userInfo.SupplierId)
			if err != nil {
				return nil, err, http.StatusInternalServerError
			}
		}

		if len(cities) == 0 {
			err = fmt.Errorf("Current user does not have any warehouses assigned")
			return nil, err, http.StatusBadRequest
		}
	}

	totalPages := 0
	entries := make([]SearchResultEntry, 0)
	tries := 0
	for {
		tries++

		hits, totalPages_, err := mh.es.search(&searchQuery, ctx)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		}

		entries_, err := mh.getResponseEntries(ctx, hits, userInfo, searchQuery.CityID, searchQuery.InStockOnly, searchQuery.Supplier)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		}

		entries = append(entries, entries_...)
		totalPages = totalPages_

		if len(entries) > itemsPerPage/10 || searchQuery.Page >= totalPages-1 {
			break
		}
		searchQuery.Page++
	}

	log.Info("Have done ", tries, " queries to get ", len(entries), " product entries")

	return &SearchResults{userInfo, cities, searchQuery.Page, totalPages, entries}, nil, http.StatusOK
}

func (mh *MethodHandlers) getResponseEntries(ctx context.Context, hits []interface{}, userInfo UserInfo, cityId int, inStockOnly bool, supplier string) ([]SearchResultEntry, error) {

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

	products, err := mh.prodDB.getProductEntries(ctx, ids, products_score, userInfo, cityId, inStockOnly, supplier)

	log.Debug("Got info for ", len(products), " entries")

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

	userInfo := mh.getUserInfo(r)
	cities, err := mh.prodDB.getUserConsigneeCities(r.Context(), userInfo)
	if len(cities) == 0 && userInfo.SupplierId != 0 {
		cities, err = mh.prodDB.getSupplierCities(r.Context(), userInfo.SupplierId)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-CSRF-Token", csrf.Token(r))
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		UserInfo
		Cities []City `json:"cities"`
	}{userInfo, cities})

	log.Info(userInfo)

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil

}

func (mh *MethodHandlers) getCounterpartsHandler(w http.ResponseWriter, r *http.Request) error {

	var filter CounterpartsFilter
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err := decoder.Decode(&filter, r.URL.Query())
	if err != nil {
		err = fmt.Errorf("Failed to decode filter: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	userInfo := mh.getUserInfo(r)

	if !userInfo.Admin && !userInfo.Staff {
		log.Error("Insufficient privileges to get list of counterparts")
		http.Error(w, "Inssuficient privileges", http.StatusUnauthorized)
		return err
	}

	log.Info("Getting list of counterparts, filter ", filter)

	counterparts, err := mh.prodDB.getCounterparts(r.Context(), userInfo, filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		Counterparts interface{} `json:"counterparts"`
	}{counterparts})

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}

func (mh *MethodHandlers) getCounterpartsExcelHandler(w http.ResponseWriter, r *http.Request) error {

	var filter CounterpartsFilter
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err := decoder.Decode(&filter, r.URL.Query())
	if err != nil {
		err = fmt.Errorf("Failed to decode filter: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	userInfo := mh.getUserInfo(r)

	if !userInfo.Admin && !userInfo.Staff {
		log.Error("Insufficient privileges to get excel with list of counterparts")
		http.Error(w, "This resource requires admin privileges", http.StatusUnauthorized)
		return err
	}

	log.Info("Getting list of counterparts, filter ", filter)

	counterparts, err := mh.prodDB.getCounterparts(r.Context(), userInfo, filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	file, err := os.CreateTemp("/tmp", "*.xlsx")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	defer os.Remove(file.Name())

	xls := excelize.NewFile()
	streamWriter, err := xls.NewStreamWriter("Sheet1")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	columnNames := []interface{}{
		excelize.Cell{Value: "#"},
		excelize.Cell{Value: "Название компании"},
		excelize.Cell{Value: "Роль"},
		excelize.Cell{Value: "ИНН"},
		excelize.Cell{Value: "КПП"},
		excelize.Cell{Value: "ОГРН"},
		excelize.Cell{Value: "Юридический адрес"},
		excelize.Cell{Value: "Фактический адрес"},
		excelize.Cell{Value: "Директор"},
		excelize.Cell{Value: "Контактное лицо"},
		excelize.Cell{Value: "Телефон"},
		excelize.Cell{Value: "Банк"},
		excelize.Cell{Value: "БИК"},
		excelize.Cell{Value: "Корр. счёт"},
		excelize.Cell{Value: "Рассчётный счёт"},
		excelize.Cell{Value: "Дополнительные поля"},
		excelize.Cell{Value: "Телефон банка"},
		excelize.Cell{Value: "Счёт"},
		excelize.Cell{Value: "IBAN"},
		excelize.Cell{Value: "SWIFT"},
		excelize.Cell{Value: "Страна"},
		excelize.Cell{Value: "Город"},
		excelize.Cell{Value: "E-mail"},
		excelize.Cell{Value: "Сайт"},
		excelize.Cell{Value: "Телефон"},
		excelize.Cell{Value: "Имя"},
		excelize.Cell{Value: "Телефон"},
		excelize.Cell{Value: "E-Mail"},
	}
	for i, columnName := range columnNames {
		cellWidth := utf8.RuneCountInString(columnName.(excelize.Cell).Value.(string)) + 2 // + 2 for margin
		if cellWidth < 6 {
			cellWidth = 6
		}
		streamWriter.SetColWidth(i+1, i+1, float64(cellWidth))
	}

	streamWriter.MergeCell("A1", "T1")
	streamWriter.SetRow("U1", []interface{}{
		excelize.Cell{Value: "Поставщик"},
	})

	streamWriter.MergeCell("U1", "Y1")
	streamWriter.SetRow("Z1", []interface{}{
		excelize.Cell{Value: "Администратор"},
	})

	streamWriter.MergeCell("Z1", "AB1")

	streamWriter.SetRow("A2", columnNames)

	row := 3
	for _, cp := range counterparts {

		streamWriter.SetRow(fmt.Sprintf("A%v", row), []interface{}{
			excelize.Cell{Value: cp["id"]},
			excelize.Cell{Value: cp["name"]},
			excelize.Cell{Value: cp["role"]},
			excelize.Cell{Value: cp["inn"]},
			excelize.Cell{Value: cp["kpp"]},
			excelize.Cell{Value: cp["ogrn"]},
			excelize.Cell{Value: cp["address"]},
			excelize.Cell{Value: cp["actual_address"]},
			excelize.Cell{Value: cp["director_name"]},
			excelize.Cell{Value: cp["contact_name"]},
			excelize.Cell{Value: cp["phone"]},
			excelize.Cell{Value: cp["bank"]},
			excelize.Cell{Value: cp["bik"]},
			excelize.Cell{Value: cp["corr_account"]},
			excelize.Cell{Value: cp["pay_account"]},
			excelize.Cell{Value: cp["extra_data"]},
			excelize.Cell{Value: cp["bank_phone"]},
			excelize.Cell{Value: cp["account"]},
			excelize.Cell{Value: cp["IBAN"]},
			excelize.Cell{Value: cp["SWIFT"]},
			excelize.Cell{Value: cp["country"]},
			excelize.Cell{Value: cp["city"]},
			excelize.Cell{Value: cp["seller_email"]},
			excelize.Cell{Value: cp["seller_site"]},
			excelize.Cell{Value: cp["seller_phone"]},
		})

		if cp["admins"] != nil {
			admins := cp["admins"].(map[string]interface{})
			rowStart := row
			for _, a := range admins {
				admin := a.(map[string]interface{})

				streamWriter.SetRow(fmt.Sprintf("Z%v", row), []interface{}{
					excelize.Cell{Value: toString(admin["name"])},
					excelize.Cell{Value: toString(admin["phone"])},
					excelize.Cell{Value: toString(admin["email"])},
				})
				row++
			}

			adminsLen := len(admins)

			if adminsLen > 1 {

				for col := 'A'; col <= 'Y'; col++ {
					streamWriter.MergeCell(fmt.Sprintf("%c%v", col, rowStart), fmt.Sprintf("%c%v", col, row-1))
				}
			}
		} else {

			row++
		}

	}

	err = streamWriter.Flush()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	xls.SaveAs(file.Name())

	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote("Заказы.xlsx"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, file.Name())

	return nil
}

type Orders struct {
	Orders     []OrderDetails `json:"orders"`
	Count      int            `json:"count"`
	Sum        float64        `json:"sum"`
	SumWithTax float64        `json:"sum_with_tax"`
}

// @Summary List orders
// @Description Get orders list
// @Tags orders
// @Produce  json
// @Param start query string false "Start of the period used to filter orders, in datetime format (e.g. 2021-10-23T21:00:00.000Z)"
// @Param end query string false "End of the period used to filter orders, in datetime format (e.g. 2021-10-24T20:59:59.999Z)"
// @Param dateColumn query string false "Date used to filter orders" Enums(date_ordered, date_closed)
// @Param text query string false "Query used to filter orders, might be customer name, order number or buyer name"
// @Param itemsPerPage query int false "Page size" default(10) minimum(1) maximum(10)
// @Param page query int false "Page number" default(0)
// @Param selectedStatuses[] query []int false "Order status (Создан 1, В обработке 2, На согласовании 3, На сборке 10, В пути 21, Доставлен 15, Приёмка 20, Принят 22, Завершён 24, Отказ/Не согласован 4)"
// @Success 200 {object} Orders
// @Router /orders/ [get]
func (mh *MethodHandlers) getOrdersHandler(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.getUserInfo(r)

	var ordersFilter OrdersFilter
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err := decoder.Decode(&ordersFilter, r.URL.Query())
	if err != nil {
		err = fmt.Errorf("Failed to decode filter: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	orders, err, code := mh.getOrders(r.Context(), userInfo, ordersFilter)
	if err != nil {
		http.Error(w, err.Error(), code)
		return err
	}

	err = json.NewEncoder(w).Encode(orders)

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}

func (mh *MethodHandlers) getOrders(ctx context.Context, userInfo UserInfo, ordersFilter OrdersFilter) (*Orders, error, int) {

	if ordersFilter.ItemsPerPage <= 0 {
		ordersFilter.ItemsPerPage = 10
	}
	if ordersFilter.ItemsPerPage > 1000 {
		ordersFilter.ItemsPerPage = 1000
	}

	log.Info("Getting list of orders, filter ", ordersFilter)

	ordersList, err := mh.prodDB.getOrders(ctx, userInfo, ordersFilter)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}

	orders := Orders{Orders: ordersList}

	if ordersFilter.Page == 0 {
		orders.Count, orders.Sum, orders.SumWithTax, err = mh.prodDB.getOrdersSum(ctx, userInfo, ordersFilter)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		}
	}
	return &orders, nil, http.StatusOK

}

// @Summary List order lines
// @Description Get order itemslist
// @Param orderId path int true "Order Id"
// @Tags order
// @Produce  json
// @Success 200 {object} OrderLines
// @Router /orders/{orderId} [get]
func (mh *MethodHandlers) getOrderHandler(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.getUserInfo(r)

	params := mux.Vars(r)
	orderId_ := params["orderId"]
	orderId, err := strconv.Atoi(orderId_)
	if err != nil {
		err = fmt.Errorf("Failed to determin requested order ID: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	orderDetails, err := mh.prodDB.getOrder(r.Context(), userInfo, orderId)
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

	userInfo := mh.getUserInfo(r)
	var ordersFilter OrdersFilter
	err := schema.NewDecoder().Decode(&ordersFilter, r.URL.Query())
	if err != nil {
		err = fmt.Errorf("Failed to decode filter: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	file, err := os.CreateTemp("/tmp", "*.xlsx")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	defer os.Remove(file.Name())

	err, code := mh.getOrdersExcel(r.Context(), userInfo, ordersFilter, file.Name())
	if err != nil {
		http.Error(w, err.Error(), code)
		return err
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote("Заказы.xlsx"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, file.Name())

	return nil
}

func (mh *MethodHandlers) getOrdersExcel(ctx context.Context, userInfo UserInfo, ordersFilter OrdersFilter, fileName string) (err error, code int) {

	if userInfo.SupplierId != 0 {
		err := fmt.Errorf("This resource is not available for suppliers")
		return err, http.StatusUnauthorized
	}

	ordersFilter.Page = 0
	ordersFilter.ItemsPerPage = 0

	xls := excelize.NewFile()
	streamWriter, err := xls.NewStreamWriter("Sheet1")
	if err != nil {
		return err, http.StatusInternalServerError
	}

	//set column width
	columnNames := []interface{}{
		excelize.Cell{Value: "ID  "},
		excelize.Cell{Value: "Номер заказа"},
		excelize.Cell{Value: "Дата заказа"},
		excelize.Cell{Value: "Дата согласования"},
		excelize.Cell{Value: "Продавец"},
		excelize.Cell{Value: "Адрес"},
		excelize.Cell{Value: "ИНН/КПП продавца"},
		excelize.Cell{Value: "Грузоотправитель"},
		excelize.Cell{Value: "Адрес грузоотправителя"},
		excelize.Cell{Value: "Грузополучатель"},
		excelize.Cell{Value: "Адрес грузополучателя"},
		excelize.Cell{Value: "Покупатель"},
		excelize.Cell{Value: "Адрес покупателя"},
		excelize.Cell{Value: "ИНН поставщика"},
		excelize.Cell{Value: "КПП поставщика"},
		excelize.Cell{Value: "ИНН покупателя"},
		excelize.Cell{Value: "КПП покупателя"},
		excelize.Cell{Value: "Код товара/работ, услуг"},
		excelize.Cell{Value: "Артикул"},
		excelize.Cell{Value: "Наименование товара"},
		excelize.Cell{Value: "Единица обозначения - условное обозначение (национальное)"},
		excelize.Cell{Value: "Количество (объём)"},
		excelize.Cell{Value: "Цена (тариф) за единицу измерения"},
		excelize.Cell{Value: "Стоимость товаров (работ, услуг), имущественных прав без налога - всего"},
		excelize.Cell{Value: "Налоговая ставка"},
		excelize.Cell{Value: "Сумма налога, предъявляемая покупателю"},
		excelize.Cell{Value: "Стоимость товаров (работ, услуг), имущественных прав с налогом - всего"},
		excelize.Cell{Value: "Дата поставки (предполагаемая)"},
		excelize.Cell{Value: "Дата отгрузки"},
		excelize.Cell{Value: "Время отгрузки"},
		excelize.Cell{Value: "Дата передачи"},
		excelize.Cell{Value: "Время передачи"},
		excelize.Cell{Value: "Дата приёмки"},
		excelize.Cell{Value: "Время приёмки"},
		excelize.Cell{Value: "Статус"},
		excelize.Cell{Value: "Имя              "},
		excelize.Cell{Value: "Электронная почта"},
	}
	for i, columnName := range columnNames {
		cellWidth := utf8.RuneCountInString(columnName.(excelize.Cell).Value.(string)) + 2 // + 2 for margin
		streamWriter.SetColWidth(i+1, i+1, float64(cellWidth))
	}

	streamWriter.SetRow("A1", []interface{}{
		excelize.Cell{},
		excelize.Cell{Value: "[8]"},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{Value: "(2)"},
		excelize.Cell{Value: "(2а)"},
		excelize.Cell{Value: "(2б)"},
		excelize.Cell{Value: "(3)"},
		excelize.Cell{},
		excelize.Cell{Value: "(4)"},
		excelize.Cell{},
		excelize.Cell{Value: "(6) и [19]"},
		excelize.Cell{Value: "(6а)"},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{Value: "(6б)"},
		excelize.Cell{},
		excelize.Cell{Value: "Табличная часть УПД"},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{Value: "Статус заказа \"В пути\""},
		excelize.Cell{},
		excelize.Cell{Value: "Статус заказа \"Доставлен\""},
		excelize.Cell{},
		excelize.Cell{Value: "Статус заказа \"Принят\""},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{Value: "Закупщик"},
		excelize.Cell{},
	})

	streamWriter.MergeCell("B1", "C1")
	streamWriter.MergeCell("H1", "I1")
	streamWriter.MergeCell("J1", "K1")
	streamWriter.MergeCell("P1", "Q1")
	streamWriter.MergeCell("R1", "AA1")
	streamWriter.MergeCell("AC1", "AD1")
	streamWriter.MergeCell("AE1", "AF1")
	streamWriter.MergeCell("AG1", "AH1")
	streamWriter.MergeCell("AJ1", "AK1")

	streamWriter.SetRow("A2", columnNames)

	orders, err := mh.prodDB.getOrders(ctx, userInfo, ordersFilter)
	if err != nil {
		return err, http.StatusInternalServerError
	}

	row := 3
	for _, order := range orders {
		orderStartRow := row

		orderDetails, err := mh.prodDB.getOrder(ctx, userInfo, order.Id)
		if err != nil {
			return err, http.StatusInternalServerError
		}

		for i := 0; i < len(orderDetails); i++ {
			orderDetail := orderDetails[i]

			streamWriter.SetRow(fmt.Sprintf("A%v", row), []interface{}{
				excelize.Cell{Value: order.Id},
				excelize.Cell{Value: order.ContractorNumber},
				excelize.Cell{Value: toDateString(order.OrderedDate)},
				excelize.Cell{Value: toDateString(order.ClosedDate)},
				excelize.Cell{Value: "Общество с ограниченной ответственностью \"Центр Промышленных Закупок\""},
				excelize.Cell{Value: "127299, г. Москва, ул. Клары Цеткин, д. 2, помещ. 138"},
				excelize.Cell{Value: "3528136252/771301001"},
				excelize.Cell{Value: order.SellerName},
				excelize.Cell{Value: order.SellerAddress},
				excelize.Cell{Value: order.CustomerName},
				excelize.Cell{Value: order.CustomerAddress},
				excelize.Cell{Value: order.CustomerName},
				excelize.Cell{Value: order.CustomerAddress},
				excelize.Cell{Value: order.SellerInn},
				excelize.Cell{Value: order.SellerKpp},
				excelize.Cell{Value: order.CustomerInn},
				excelize.Cell{Value: order.CustomerKpp},

				excelize.Cell{Value: orderDetail.ProductId},
				excelize.Cell{Value: orderDetail.Code},
				excelize.Cell{Value: orderDetail.Name},
				excelize.Cell{Value: "Шт"},
				excelize.Cell{Value: orderDetail.Count},
				excelize.Cell{Value: orderDetail.Price},
				excelize.Cell{Value: orderDetail.Sum},
				excelize.Cell{Value: orderDetail.Nds},
				excelize.Cell{Value: orderDetail.Tax},
				excelize.Cell{Value: orderDetail.Tax + orderDetail.Sum},

				excelize.Cell{Value: toDateString(order.ShippingDateEst)},
				excelize.Cell{Value: toDateString(order.ShippedDate)},
				excelize.Cell{Value: toTimeString(order.ShippedDate)},
				excelize.Cell{Value: toDateString(order.DeliveredDate)},
				excelize.Cell{Value: toTimeString(order.DeliveredDate)},
				excelize.Cell{Value: toDateString(order.AcceptedDate)},
				excelize.Cell{Value: toTimeString(order.AcceptedDate)},
				excelize.Cell{Value: order.Status},
				excelize.Cell{Value: order.Buyer},
				excelize.Cell{Value: order.BuyerEmail},
			})

			row++
		}

		if len(orderDetails) > 1 {

			for col := 'A'; col <= 'Q'; col++ {
				err = streamWriter.MergeCell(fmt.Sprintf("%c%v", col, orderStartRow), fmt.Sprintf("%c%v", col, row-1))
				if err != nil {
					return err, http.StatusInternalServerError
				}
			}

			for _, col := range []string{"AB", "AC", "AD", "AE", "AF", "AG", "AH", "AI", "AJ", "AK"} {
				err = streamWriter.MergeCell(fmt.Sprintf("%s%v", col, orderStartRow), fmt.Sprintf("%s%v", col, row-1))
				if err != nil {
					return err, http.StatusInternalServerError
				}
			}
		}

	}

	err = streamWriter.Flush()
	if err != nil {
		return err, http.StatusInternalServerError
	}
	xls.SaveAs(fileName)

	return nil, http.StatusOK
}

func (mh *MethodHandlers) getCurrentUserSI(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.getUserInfo(r)
	cities, err := mh.prodDB.getUserConsigneeCities(r.Context(), userInfo)

	if len(cities) == 0 && userInfo.SupplierId != 0 {
		cities, err = mh.prodDB.getSupplierCities(r.Context(), userInfo.SupplierId)
	}

	compareCount, err := mh.prodDB.getCompareItemsCount(r.Context(), userInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		UserInfo
		Cities            []City `json:"cities"`
		CompareItemsCount int    `json:"compareItemsCount"`
	}{userInfo, cities, compareCount})

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil

}

func (mh *MethodHandlers) getCartContent(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.getUserInfo(r)
	cartNumbers, err := mh.prodDB.getCartNumbers(r.Context(), userInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	cartItems, err := mh.prodDB.getCartItems(r.Context(), userInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		CartNumbers
		CartItems map[int]CartItem `json:"cartItems"`
	}{*cartNumbers, cartItems})

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil

}

func (mh *MethodHandlers) getApiCredentials(ctx context.Context, userInfo UserInfo) (ac *ApiCredentials, err error, code int) {
	if !userInfo.CompanyAdmin && !userInfo.Admin {
		return nil, fmt.Errorf("This resource requires company admin privileges"), http.StatusUnauthorized
	}

	apiCreds, err := mh.crutchDB.getApiCredentials(ctx, userInfo)

	return apiCreds, err, http.StatusOK
}

func (mh *MethodHandlers) getApiCredentialsHandler(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.getUserInfo(r)

	apiCreds, err, code := mh.getApiCredentials(r.Context(), userInfo)

	if err == nil {

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(apiCreds)
		if err != nil {
			err = fmt.Errorf("Error while preparing json reponse: %v", err)
			code = http.StatusInternalServerError
		}
	}

	if err != nil {
		http.Error(w, err.Error(), code)
		return err
	}

	return err
}

type apiCredParams struct {
	Enabled  *bool `json:"enabled"`
	Password *bool `json:"password"`
}

func (mh *MethodHandlers) putApiCredentials(ctx context.Context, userInfo UserInfo, params apiCredParams) (apiCreds *ApiCredentials, err error, code int) {
	if !userInfo.CompanyAdmin && !userInfo.Admin {
		return nil, fmt.Errorf("This resource requires company admin privileges"), http.StatusUnauthorized
	}

	if params.Enabled != nil {
		apiCreds, err = mh.crutchDB.setApiCredentialsEnabled(ctx, userInfo, *params.Enabled)
	}

	if params.Password != nil {
		apiCreds, err = mh.crutchDB.updateApiCredentialsPassword(ctx, userInfo)
	}

	if err != nil {
		return nil, err, http.StatusInternalServerError
	}

	return apiCreds, nil, http.StatusOK
}

func (mh *MethodHandlers) putApiCredentialsHandler(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.getUserInfo(r)

	params := apiCredParams{}

	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		err = fmt.Errorf("Failed to decode request body - %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	apiCreds, err, code := mh.putApiCredentials(r.Context(), userInfo, params)
	if err != nil {
		http.Error(w, err.Error(), code)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(apiCreds)

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}
