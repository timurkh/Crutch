package main

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/logrusadapter"
	"github.com/jackc/pgx/v4/pgxpool"
)

type CounterpartsFilter struct {
	Start      time.Time `schema:"start"`
	End        time.Time `schema:"end"`
	Text       string    `schema:"text"`
	Role       int       `schema:"role"`
	HaveOrders bool      `schema:"haveOrders"`
	Verified   bool      `schema:"verified"`
}

type OrdersFilter struct {
	Start            time.Time `schema:"start"`
	End              time.Time `schema:"end"`
	Text             string    `schema:"text"`
	SelectedStatuses []int     `schema:"selectedStatuses[]"`
	DateColumn       string    `schema:"dateColumn"`
	Page             int       `schema:"page"`
	ItemsPerPage     int       `schema:"itemsPerPage"`
}

type ProdDBHelper struct {
	pool *pgxpool.Pool
}

func initProdDBHelper(host string, user string, password string, database string) (*ProdDBHelper, error) {
	url := "postgres://" + user + ":" + password + "@" + host + "/" + database
	conf, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	conf.ConnConfig.Logger = logrusadapter.NewLogger(log)
	conf.ConnConfig.LogLevel = pgx.LogLevelTrace

	pool, err := pgxpool.ConnectConfig(context.Background(), conf)
	if err != nil {
		return nil, err
	}

	db := ProdDBHelper{pool}
	return &db, nil
}

type UserDBInfo struct {
	first_name       string
	last_name        string
	email            string
	is_superuser     bool
	is_staff         bool
	is_company_admin bool
	can_read_orders  bool
	can_read_buyers  bool
	can_read_sellers bool
	verified         bool
	blocked          bool
	contractor_id    int
	contractor_name  string
	supplier_id      int
	supplier_name    string
}

func (db *ProdDBHelper) getUserInfo(userId int) (*UserDBInfo, error) {
	ui := UserDBInfo{}
	err := db.pool.QueryRow(context.Background(), `
		SELECT first_name, 
			last_name, 
			email, 
			is_superuser, 
			is_staff,
			company_admin,
			COALESCE( can_read_orders, FALSE) can_read_orders,
			COALESCE( can_read_buyers, FALSE) can_read_buyers,
			COALESCE( can_read_sellers, FALSE) can_read_sellers,
			verified, 
			blocked,
			COALESCE( current_contractor_id, 0),
			COALESCE( cc.name, ''),
			COALESCE( supplier_id, 0),
			COALESCE( sp.name, '')
		FROM 
			core_user cu 
			LEFT JOIN (SELECT user_id, TRUE as can_read_orders FROM core_user_user_permissions up WHERE permission_id=1067) ro ON (ro.user_id=cu.id AND cu.is_staff) 
			LEFT JOIN (SELECT user_id, TRUE as can_read_buyers FROM core_user_user_permissions up WHERE permission_id=286) rb ON (rb.user_id=cu.id AND cu.is_staff) 
			LEFT JOIN (SELECT user_id, TRUE as can_read_sellers FROM core_user_user_permissions up WHERE permission_id=678) rs ON (rs.user_id=cu.id AND cu.is_staff) 
			LEFT JOIN company_company cc ON (cc.object_id=cu.current_contractor_id AND cc.content_type_id=79)
			LEFT JOIN company_company sp ON (sp.object_id=cu.supplier_id AND sp.content_type_id=186)
		WHERE cu.id =$1`, userId).Scan(&ui.first_name, &ui.last_name, &ui.email, &ui.is_superuser, &ui.is_staff, &ui.is_company_admin, &ui.can_read_orders, &ui.can_read_buyers, &ui.can_read_sellers, &ui.verified, &ui.blocked, &ui.contractor_id, &ui.contractor_name, &ui.supplier_id, &ui.supplier_name)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve user info: %v", err)
	}

	return &ui, nil

}

func (db *ProdDBHelper) getUserConsigneeCities(ctx context.Context, userInfo UserInfo) (cities []City, err error) {

	var rows pgx.Rows

	if userInfo.Admin {
		rows, _ = db.pool.Query(ctx, "SELECT distinct id, city FROM company_city ORDER BY city")
	} else {
		rows, _ = db.pool.Query(ctx, "SELECT distinct city_id, city FROM core_user cu JOIN core_user_contractors cuc on cu.id = cuc.user_id join consignee_consignee con using(contractor_id) join company_city com on com.id = con.city_id where cuc.user_id=$1 ORDER BY city", userInfo.Id)
	}

	cities = make([]City, 0)
	for rows.Next() {
		var city City
		err := rows.Scan(&city.Id, &city.Name)
		if err != nil {
			return nil, err
		}
		cities = append(cities, city)
	}

	return cities, rows.Err()
}

func (db *ProdDBHelper) getSupplierCities(ctx context.Context, supplierId int) (cities []City, err error) {

	var rows pgx.Rows

	rows, _ = db.pool.Query(ctx, `
		SELECT DISTINCT dc.city_id, city 
		FROM supplier_warehouse_delivery_cities dc 
			JOIN supplier_warehouse sw ON sw.id = dc.warehouse_id 
			JOIN company_city c ON c.id=dc.city_id 
		WHERE supplier_id=$1`, supplierId)

	cities = make([]City, 0)
	for rows.Next() {
		var city City
		err := rows.Scan(&city.Id, &city.Name)
		if err != nil {
			return nil, err
		}
		cities = append(cities, city)
	}

	return cities, rows.Err()
}

type SearchResultEntry struct {
	Id             int     `json:"id"`
	Category       string  `json:"category"`
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Rest           float64 `json:"rest"`
	Price          float64 `json:"price"`
	Supplier       string  `json:"supplier"`
	Image          string  `json:"image"`
	ModificationId int     `json:"modification_id"`
	WarehouseId    int     `json:"warehouse_id"`
	Score          float64 `json:"score"`
}

func (db *ProdDBHelper) getProductEntries(ctx context.Context, product_ids []int, products_score map[int]float64, userInfo UserInfo, city_id int, inStockOnly bool, supplier string) (products []SearchResultEntry, err error) {

	args := []interface{}{product_ids}

	supplier_warehouses := ""
	if userInfo.Admin || userInfo.SupplierId != 0 {
		if city_id > 0 {
			args = append(args, city_id)
			supplier_warehouses = `
			AND pr.warehouse_id IN (
			SELECT sw.id 
			FROM supplier_warehouse sw 
				INNER JOIN supplier_warehouse_delivery_cities swc ON (sw.id = swc.warehouse_id) 
			WHERE sw.is_visible = true  
				AND swc.city_id = $` + strconv.Itoa(len(args))
			supplier_warehouses += `)`
		}
	} else {
		supplier_warehouses = `
		AND pr.warehouse_id IN (
		SELECT sw.id 
		FROM supplier_warehouse sw 
			INNER JOIN supplier_warehouse_delivery_cities swc ON (sw.id = swc.warehouse_id) 
		WHERE sw.is_visible = true  
		`
		args = append(args, userInfo.Id)
		client_cities := `
		SELECT DISTINCT city_id 
		FROM core_user_contractors cuc 
			JOIN consignee_consignee con USING(contractor_id) 
			JOIN company_city com on com.id = con.city_id 
		WHERE cuc.user_id=$` + strconv.Itoa(len(args))

		supplier_warehouses += `	AND swc.city_id IN (` + client_cities + `)`

		if city_id > 0 {
			args = append(args, city_id)
			supplier_warehouses = supplier_warehouses + " AND swc.city_id=$" + strconv.Itoa(len(args))
		}

		supplier_warehouses += `)`
	}

	product_quantity := `
	SELECT 
		pp.id,
		SUM(pr.rest) AS rest,
		ordering
	FROM
		product_product pp
		JOIN product_modification pm ON ( pp.id = pm.product_id )
		`
	if userInfo.SupplierId != 0 || userInfo.Admin {
		product_quantity += `LEFT `
	}

	product_quantity += `JOIN product_rest  pr ON ( pm.id = pr.modification_id )
		JOIN (SELECT * FROM unnest($1::int[]) WITH ORDINALITY) x (id, ordering) ON (pp.id = x.id)
	WHERE
		pp.deleted = false
		AND pm.deleted = false
		AND pp.is_reference = false
		AND pp.b_placement_state = 'placed'
	`
	if !userInfo.Admin && userInfo.SupplierId == 0 {
		product_quantity += `AND pp.category_id IS NOT NULL
		`
	}
	product_quantity += `AND pp.hidden = false  
	` + supplier_warehouses + `
	GROUP BY pp.id, ordering`

	product_modifications := `
	SELECT DISTINCT ON (pp.id)
		pp.id,
		pm.id AS modification_id,
		pr.warehouse_id
	FROM
		product_product pp
		JOIN product_modification pm ON ( pp.id = pm.product_id )
		`
	if userInfo.SupplierId != 0 || userInfo.Admin {
		product_modifications += `LEFT `
	}

	product_modifications += `JOIN product_rest  pr ON ( pm.id = pr.modification_id )
	WHERE
		pp.id = ANY($1)
		AND pp.deleted = false
		AND pm.deleted = false
		AND pp.is_reference = false
		AND pp.b_placement_state = 'placed'
		`
	if !userInfo.Admin && userInfo.SupplierId == 0 {
		product_modifications += `AND pp.category_id IS NOT NULL
		`
	}
	product_modifications += `AND pp.hidden = false  
	` + supplier_warehouses + `
	ORDER BY pp.id, pr.rest DESC`

	query := `
	SELECT
		pp.id,
		pc.name as category_name,
		pp.name,
		pp.code,
		pp.description,
		pr.rest,
		pp.product_price,
		cc.name as supplier,
		COALESCE(pi.image, ''),
		pm.modification_id,
		pm.warehouse_id
	FROM 
		( ` + product_quantity + ` 
			) pr 
		JOIN (` + product_modifications + `) pm USING (id)
		JOIN product_product pp USING (id) 
		`
	if userInfo.Admin || userInfo.SupplierId != 0 {
		query += `LEFT `
	}
	query += `JOIN product_category pc ON ( pp.category_id = pc.id )
		LEFT JOIN product_suppliercategory psc ON (pp.supplier_category_id = psc.id)
		LEFT JOIN company_company cc ON (cc.object_id=supplier_id AND content_type_id=186)
		LEFT JOIN (
			SELECT DISTINCT ON (pm.product_id) pi.image, pm.product_id
			FROM product_image pi
				JOIN product_modification pm ON ( pi.modification_id = pm.id )
				WHERE pi.image > '' AND pm.product_id=ANY($1) AND pm.deleted = false 
				ORDER BY pm.product_id, pi.is_base DESC, pi.position ASC, pi.id ASC
		) pi ON pi.product_id = pp.id
	WHERE 
		(pc.hidden = false OR pc.hidden IS NULL)
		`
	if !userInfo.Admin && userInfo.SupplierId == 0 || inStockOnly {
		query += `AND (NOT pr.rest = 0.0`

		if !inStockOnly {
			query += " OR pp.enable_preorder = true"
		}
		query += ")"
	}

	if userInfo.SupplierId != 0 {
		args = append(args, userInfo.SupplierId)
		query += " AND supplier_id=$" + strconv.Itoa(len(args))
	} else if supplier != "" {
		args = append(args, "%"+supplier+"%")
		query += " AND cc.name ILIKE $" + strconv.Itoa(len(args))
	}

	query += " ORDER BY ordering"

	rows, _ := db.pool.Query(ctx, query, args...)

	products = make([]SearchResultEntry, 0)
	for rows.Next() {
		var category, name, code, description, supplier string
		var price float64
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		id := toInt(values[0])

		if values[1] != nil {
			category = toString(values[1])
		}
		name = values[2].(string)
		if values[3] != nil {
			code = toString(values[3])
		}
		if values[4] != nil {
			description = toString(values[4])
		}

		p := values[6].(pgtype.Numeric)
		p.AssignTo(&price)

		if values[7] != nil {
			supplier = toString(values[7])
		}

		entry := SearchResultEntry{
			id,
			category,
			code,
			name,
			description,
			toFloat(values[5]),
			price,
			supplier,
			toString(values[8]),
			toInt(values[9]),
			toInt(values[10]),
			products_score[id],
		}
		products = append(products, entry)
	}

	return products, rows.Err()
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func toFloat(v interface{}) float64 {
	if v == nil {
		return 0
	}
	return v.(float64)
}

func toInt(v interface{}) int {
	if v == nil {
		return 0
	}
	return int(v.(int32))
}

func toDateString(v *time.Time) string {
	if v != nil {
		return v.Format("2006-01-02")
	}
	return ""
}

func toTime(v interface{}) *time.Time {
	if v != nil {
		t := v.(time.Time)
		return &t
	}
	return nil
}

func toTimeString(v *time.Time) string {
	if v != nil {
		return v.Format("15:04:05")
	}
	return ""
}

func (db *ProdDBHelper) getCounterparts(ctx context.Context, userInfo UserInfo, filter CounterpartsFilter) (counterparts []map[string]interface{}, err error) {

	args := make([]interface{}, 0)
	dateFilter := ""

	// filter by date
	args = append(args, filter.Start)
	dateFilter += " AND order_order.date_ordered>$" + strconv.Itoa(len(args))

	args = append(args, filter.End)
	dateFilter += " AND order_order.date_ordered<$" + strconv.Itoa(len(args))

	query := `
		SELECT * FROM (
			SELECT 
				cc.id,
				content_type_id AS role_id,
				name, 
				inn,
				kpp,
				jur_address AS address,
				CASE WHEN content_type_id=79 THEN 'Покупатель' 
					WHEN content_type_id=186 THEN 'Поставщик'
					ELSE '-'
				END AS role, 
				CASE WHEN content_type_id=79 THEN contractor_user_count 
					WHEN content_type_id=186 THEN supplier_user_count
					ELSE 0 
				END AS user_count,
				CASE 
					WHEN content_type_id=79 THEN c.created_at
					WHEN content_type_id=186 THEN s.created_at
					ELSE NULL 
				END AS date_created,
				CASE WHEN content_type_id=79 THEN contractor_date_verified 
					WHEN content_type_id=186 THEN supplier_date_verified
					ELSE NULL 
				END AS date_verified,
				CASE WHEN content_type_id=79 THEN contractor_last_login 
					WHEN content_type_id=186 THEN supplier_last_login
					ELSE NULL 
				END AS last_login,
				CASE WHEN content_type_id=79 THEN contractor_order_count 
					WHEN content_type_id=186 THEN supplier_order_count
					ELSE NULL 
				END AS order_count,
				ogrn,
				actual_address,
				director_name,
				contact_name,
				cc.phone,
				bank,
				bik,
				corr_account,
				pay_account,
				extra_data,
				bank_phone,
				account,
				"IBAN",
				"SWIFT",
				country,
				city,
				ss.email,
				ss.site,
				ss.phone,
				CASE WHEN content_type_id=79 THEN contractor_admin
					WHEN content_type_id=186 THEN supplier_admin
					ELSE NULL
				END AS admin
			FROM company_company cc
				LEFT JOIN (SELECT COUNT(*) supplier_user_count,  MIN(date_joined) supplier_date_verified, MAX(last_login) AS supplier_last_login, supplier_id 
					FROM core_user WHERE verified=TRUE AND blocked=FALSE GROUP BY supplier_id) cu 
					ON (object_id = cu.supplier_id AND content_type_id=186) 
				LEFT JOIN (SELECT COUNT (*) contractor_user_count, MIN(date_joined) contractor_date_verified, MAX(last_login) contractor_last_login, contractor_id 
					FROM core_user_contractors JOIN core_user cu ON (cu.id = user_id AND verified=TRUE and blocked=FALSE) GROUP BY contractor_id) cuc 
					ON (object_id=cuc.contractor_id AND content_type_id=79)
				LEFT JOIN (SELECT COUNT (*) supplier_order_count, supplier_id FROM order_order WHERE id NOT IN (1, 17, 18) `
	query += dateFilter
	query += ` GROUP BY supplier_id) so
					ON (object_id = so.supplier_id AND content_type_id=186)
				LEFT JOIN (SELECT COUNT (*) contractor_order_count, contractor_id FROM order_order WHERE id NOT IN (1, 17, 18) `
	query += dateFilter
	query += ` GROUP BY contractor_id) co
					ON (object_id = co.contractor_id AND content_type_id=79)
				LEFT JOIN supplier_supplier s ON s.id = cc.object_id and content_type_id=186
				LEFT JOIN supplier_supplierprofile ss ON cc.object_id = ss.supplier_id and content_type_id=186
				LEFT JOIN contractor_contractor c ON c.id = cc.object_id and content_type_id=79
				LEFT JOIN company_country ON company_country.id = ss.country_id
				LEFT JOIN company_city ON company_city.id = ss.city_id
				LEFT JOIN (SELECT u.supplier_id, json_object_agg (u.id, json_build_object('name', concat_ws(' ', first_name, middle_name, last_name), 'email', u.email, 'phone', u.phone)) AS supplier_admin 
					FROM core_user u 
					WHERE u.company_admin=TRUE GROUP BY u.supplier_id) cas 
					ON (object_id = cas.supplier_id AND content_type_id=186)
				LEFT JOIN (SELECT cc.contractor_id, json_object_agg(u.id, json_build_object('name', concat_ws(' ', first_name, middle_name, last_name), 'email', u.email, 'phone', u.phone)) AS contractor_admin 
					FROM core_user u JOIN core_user_contractors cc ON u.id = cc.user_id 
					WHERE u.company_admin=TRUE GROUP BY cc.contractor_id) cac 
					ON (object_id = cac.contractor_id AND content_type_id=79)
		)s 
		`
	query += ` WHERE TRUE `
	if filter.Verified {
		query += ` AND user_count>0 `
	}

	if !userInfo.Admin {
		query += " AND role_id IN (0"
		if userInfo.CanReadBuyers {
			query += ", 79"
		}
		if userInfo.CanReadSellers {
			query += ", 186"
		}
		query += ")"
	}

	// filter by text
	if filter.Text != "" {
		args = append(args, "%"+filter.Text+"%")
		n := strconv.Itoa(len(args))
		query += " AND (name ILIKE $" + n
		query += " OR inn ILIKE $" + n
		query += " OR kpp ILIKE $" + n
		query += " OR address ILIKE $" + n
		query += ")"
	}

	if filter.HaveOrders {
		query += " AND order_count > 0"
	}

	if filter.Role > 0 {
		args = append(args, filter.Role)
		query += " AND role_id = $" + strconv.Itoa(len(args))
	}

	query += ` ORDER BY id`

	rows, _ := db.pool.Query(ctx, query, args...)

	counterparts = make([]map[string]interface{}, 0)
	for rows.Next() {

		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		entry := map[string]interface{}{
			"id":             values[0],
			"role_id":        values[1],
			"name":           values[2],
			"inn":            values[3],
			"kpp":            values[4],
			"address":        values[5],
			"role":           values[6],
			"user_count":     values[7],
			"date_created":   values[8],
			"date_verified":  values[9],
			"last_login":     values[10],
			"order_count":    values[11],
			"ogrn":           toString(values[12]),
			"actual_address": values[13],
			"director_name":  toString(values[14]),
			"contact_name":   toString(values[15]),
			"phone":          toString(values[16]),
			"bank":           toString(values[17]),
			"bik":            toString(values[18]),
			"corr_account":   toString(values[19]),
			"pay_account":    toString(values[20]),
			"extra_data":     toString(values[21]),
			"account":        toString(values[22]),
			"bank_phone":     toString(values[23]),
			"IBAN":           toString(values[24]),
			"SWIFT":          toString(values[25]),
			"country":        toString(values[26]),
			"city":           toString(values[27]),
			"seller_email":   toString(values[28]),
			"seller_site":    toString(values[29]),
			"seller_phone":   toString(values[30]),
			"admins":         values[31],
		}

		counterparts = append(counterparts, entry)
	}

	return counterparts, rows.Err()
}

func (db *ProdDBHelper) ordersAccessRightsFilter(userInfo UserInfo) (string, []interface{}) {
	filterUsers := ""
	args := make([]interface{}, 0)
	if !userInfo.Admin && !(userInfo.Staff && userInfo.CanReadOrders) {

		if userInfo.SupplierId > 0 && userInfo.CompanyAdmin {
			args = append(args, userInfo.SupplierId)
			filterUsers = ` AND oo.supplier_id = $`
			filterUsers += strconv.Itoa(len(args))
			filterUsers += " AND oo.status_id NOT IN (18, 26, 23, 24, 26)"
		} else if userInfo.CompanyAdmin {

			args = append(args, userInfo.Id)
			filterUsers = ` AND oo.contractor_id in (
				SELECT contractor_id FROM core_user_contractors 
				WHERE user_id=$`
			filterUsers += strconv.Itoa(len(args)) + ")"
		} else {

			args = append(args, userInfo.Id)
			filterUsers = ` AND oo.user_id in (
				SELECT ou.id 
				FROM core_user ou 
					JOIN core_user cu ON (ou.lft <= cu.rght  AND ou.lft >= cu.lft  AND ou.tree_id = cu.tree_id) 
				WHERE cu.id=$`
			filterUsers += strconv.Itoa(len(args)) + ")"
		}
	} else {
		args = append(args, true)
		filterUsers = " AND $1"
	}

	return filterUsers, args
}

func (db *ProdDBHelper) getOrdersFilterQuery(userInfo UserInfo, ordersFilter OrdersFilter) (filter string, args []interface{}) {

	// filter by access rights
	filterUsers, args := db.ordersAccessRightsFilter(userInfo)

	// filter by date
	dateColumn := ""
	switch ordersFilter.DateColumn {
	case "date_ordered":
		dateColumn = "date_ordered"
	case "date_closed":
		dateColumn = "date_closed"
	}

	if dateColumn != "" {
		if !ordersFilter.End.IsZero() {
			args = append(args, ordersFilter.End)
			filter = " AND oo." + dateColumn + "<$" + strconv.Itoa(len(args))
		}

		if !ordersFilter.Start.IsZero() {
			args = append(args, ordersFilter.Start)
			filter += " AND oo." + dateColumn + ">$" + strconv.Itoa(len(args))
		}
	}

	// filter by text
	if ordersFilter.Text != "" {
		args = append(args, "%"+ordersFilter.Text+"%")
		filter += " AND (seller.name ILIKE $" + strconv.Itoa(len(args))
		filter += " OR customer.name ILIKE $" + strconv.Itoa(len(args))
		filter += " OR cc.name ILIKE $" + strconv.Itoa(len(args))
		filter += " OR CAST(oo.id AS Text) LIKE $" + strconv.Itoa(len(args))
		filter += " OR oo.contractor_number LIKE $" + strconv.Itoa(len(args))
		filter += " OR cu.last_name ILIKE $" + strconv.Itoa(len(args))
		filter += " OR cu.first_name ILIKE $" + strconv.Itoa(len(args))
		filter += " OR cu.middle_name ILIKE $" + strconv.Itoa(len(args)) + ")"
	}

	if len(ordersFilter.SelectedStatuses) > 0 {
		args = append(args, ordersFilter.SelectedStatuses)
		filter += " AND status_id = ANY($" + strconv.Itoa(len(args)) + ")"
	}

	filter += filterUsers

	return filter, args
}

func (db *ProdDBHelper) getOrdersSum(ctx context.Context, userInfo UserInfo, ordersFilter OrdersFilter) (count int, sum float64, sum_with_tax float64, err error) {

	queryOrders := `
		SELECT 
			COALESCE(COUNT(oo.id), 0),
			COALESCE(SUM(ov.order_sum), 0),
			COALESCE(SUM(ov.order_sum_with_tax), 0)
		FROM order_order oo 
			JOIN (
				SELECT oo.id, 
					ROUND((SUM(item_sum) * (100 - oo.on_order_coupon) / 100 - oo.on_order_coupon_fixed)::numeric, 2) order_sum,
					ROUND((SUM(item_sum + ROUND((item_sum*rate_nds/100)::numeric, 2)) * (100 - oo.on_order_coupon) / 100 - oo.on_order_coupon_fixed)::numeric, 2) order_sum_with_tax
					FROM order_order oo JOIN
					(SELECT id, order_id, rate_nds, ROUND((oi.count * (((oi.item_price - oi.coupon_fixed) * (100 - oi.coupon_percent)) / 100))::numeric, 2) item_sum
						FROM order_orderitem oi) oi ON oo.id = oi.order_id
				GROUP BY oo.id
			) ov USING (id)
			JOIN company_company seller ON (seller.object_id=oo.supplier_id AND seller.content_type_id=186)
			JOIN core_user cu ON (cu.id = oo.user_id)
			LEFT JOIN consignee_consignee cc ON (cc.id = oo.consignee_id)
			JOIN company_company customer ON (customer.object_id=oo.contractor_id AND customer.content_type_id=79)
		WHERE oo.status_id NOT IN (17) AND oo.deleted=FALSE AND seller.object_id!=1`

	// filter by access rights
	filter, args := db.getOrdersFilterQuery(userInfo, ordersFilter)

	queryOrders += filter

	/*
		if ordersFilter.ItemsPerPage > 0 {
			args = append(args, ordersFilter.ItemsPerPage)
			queryOrders = queryOrders + " LIMIT $" + strconv.Itoa(len(args))

			if ordersFilter.Page > 0 {
				args = append(args, ordersFilter.ItemsPerPage*ordersFilter.Page)
				queryOrders = queryOrders + " OFFSET $" + strconv.Itoa(len(args))
			}

		}
	*/

	err = db.pool.QueryRow(ctx, queryOrders, args...).Scan(&count, &sum, &sum_with_tax)
	return count, sum, sum_with_tax, err
}

type OrderDetails struct {
	Id                 int        `json:"id"`
	ContractorNumber   string     `json:"contractor_number"`
	Sum                float64    `json:"sum"`
	SumWithTax         float64    `json:"sum_with_tax"`
	Status             string     `json:"status"`
	OrderedDate        *time.Time `json:"ordered_date"`
	ClosedDate         *time.Time `json:"closed_date"`
	ShippingDateEst    *time.Time `json:"shipping_date_req"`
	SellerId           int        `json:"seller_id"`
	SellerName         string     `json:"seller_name"`
	SellerInn          string     `json:"seller_inn"`
	SellerKpp          string     `json:"seller_kpp"`
	SellerAddress      string     `json:"seller_address"`
	BuyerId            int        `json:"buyer_id"`
	Buyer              string     `json:"buyer"`
	CustomerId         int        `json:"customer_id"`
	CustomerName       string     `json:"customer_name"`
	CustomerInn        string     `json:"customer_inn"`
	CustomerKpp        string     `json:"customer_kpp"`
	CustomerAddress    string     `json:"customer_address"`
	ConsigneeName      string     `json:"consignee_name"`
	OnOrderCoupon      float64    `json:"on_order_coupon"`
	OnOrderCouponFixed float64    `json:"on_order_coupon_fixed"`
	ShippedDate        *time.Time `json:"shipped_date"`
	DeliveredDate      *time.Time `json:"delivered_date"`
	AcceptedDate       *time.Time `json:"accepted_date"`
	ConsigneeCity      string     `json:"consignee_city"`
	ConsigneeAddress   string     `json:"consignee_address"`
}

func (db *ProdDBHelper) getOrders(ctx context.Context, userInfo UserInfo, ordersFilter OrdersFilter) (orders []OrderDetails, err error) {

	queryOrders := `
		SELECT oo.id, 
			ov.order_sum,
			os.status,
			oo.date_ordered,
			oo.date_closed,
			oo.shipping_date,
			seller.object_id as seller_id,
			seller.name AS seller_name,
			seller.inn AS seller_inn,
			seller.kpp AS seller_kpp,
			seller.jur_address AS seller_address,
			cu.id AS buyer_id,
			cu.last_name || ' ' || cu.first_name || ' ' || cu.middle_name AS buyer,
			customer.object_id AS customer_id,
			customer.name AS customer_name,
			customer.inn AS customer_inn,
			customer.kpp AS customer_kpp,
			customer.jur_address AS customer_address,
			cc.name as consignee_name,
			oo.on_order_coupon,
			oo.on_order_coupon_fixed,
			oo.contractor_number, 
			ds.date_shipped,
			dd.date_delivered,
			da.date_accepted,
			ov.order_sum_with_tax,
			cc_city.city,
			cc.address as consignee_name
		FROM order_order oo 
			JOIN (
				SELECT oo.id, 
					ROUND((SUM(item_sum) * (100 - oo.on_order_coupon) / 100 - oo.on_order_coupon_fixed)::numeric, 2) order_sum,
					ROUND((SUM(item_sum + ROUND((item_sum*rate_nds/100)::numeric, 2)) * (100 - oo.on_order_coupon) / 100 - oo.on_order_coupon_fixed)::numeric, 2) order_sum_with_tax
					FROM order_order oo JOIN
					(SELECT id, order_id, rate_nds, ROUND((oi.count * (((oi.item_price - oi.coupon_fixed) * (100 - oi.coupon_percent)) / 100))::numeric, 2) item_sum
						FROM order_orderitem oi) oi ON oo.id = oi.order_id
				GROUP BY oo.id
			) ov USING (id)
			LEFT JOIN (
				SELECT object_id_int AS order_id, MIN(rr.date_created) AS date_shipped
				FROM reversion_version rv JOIN reversion_revision rr ON rv.revision_id = rr.id 
				WHERE content_type_id=115 and serialized_data::jsonb @> '[{"fields":{"status":21}}]'::jsonb
				GROUP BY object_id_int) ds ON ds.order_id = oo.id 
			LEFT JOIN (
				SELECT object_id_int AS order_id, MIN(rr.date_created) AS date_delivered
				FROM reversion_version rv JOIN reversion_revision rr ON rv.revision_id = rr.id 
				WHERE content_type_id=115 and serialized_data::jsonb @> '[{"fields":{"status":15}}]'::jsonb
				GROUP BY object_id_int) dd ON dd.order_id = oo.id 
			LEFT JOIN (
				SELECT object_id_int AS order_id, MIN(rr.date_created) AS date_accepted
				FROM reversion_version rv JOIN reversion_revision rr ON rv.revision_id = rr.id 
				WHERE content_type_id=115 and serialized_data::jsonb @> '[{"fields":{"status":22}}]'::jsonb
				GROUP BY object_id_int) da ON da.order_id = oo.id 
			JOIN order_orderstatus os ON (oo.status_id = os.id)
			JOIN company_company seller ON (seller.object_id=oo.supplier_id AND seller.content_type_id=186)
			JOIN core_user cu ON (cu.id = oo.user_id)
			LEFT JOIN consignee_consignee cc ON (cc.id = oo.consignee_id)
			LEFT JOIN company_city cc_city ON (cc_city.id = cc.city_id)
			JOIN company_company customer ON (customer.object_id=oo.contractor_id AND customer.content_type_id=79)
		WHERE oo.status_id NOT IN (17) AND oo.deleted = FALSE AND seller.object_id!=1`

	// filter by access rights
	filter, args := db.getOrdersFilterQuery(userInfo, ordersFilter)

	queryOrders += filter
	queryOrders += `	ORDER BY COALESCE(COALESCE(date_ordered, date_updated), date_created) DESC, contractor_number DESC`

	if ordersFilter.ItemsPerPage > 0 {
		args = append(args, ordersFilter.ItemsPerPage)
		queryOrders = queryOrders + " LIMIT $" + strconv.Itoa(len(args))

		if ordersFilter.Page > 0 {
			args = append(args, ordersFilter.ItemsPerPage*ordersFilter.Page)
			queryOrders = queryOrders + " OFFSET $" + strconv.Itoa(len(args))
		}

	}

	rows, _ := db.pool.Query(ctx, queryOrders, args...)

	orders = make([]OrderDetails, 0)
	for rows.Next() {

		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		id := int(values[0].(int32))
		var sum, sum_with_tax float64
		if values[1] != nil {
			s := values[1].(pgtype.Numeric)
			s.AssignTo(&sum)
		}
		if values[25] != nil {
			s := values[25].(pgtype.Numeric)
			s.AssignTo(&sum_with_tax)
		}

		status := values[2].(string)
		seller_id := int(values[6].(int32))
		seller_name := toString(values[7])
		seller_inn := toString(values[8])
		seller_kpp := toString(values[9])
		seller_address := toString(values[10])
		buyer_id := int(values[11].(int32))
		buyer := toString(values[12])
		customer_id := int(values[13].(int32))
		customer_name := toString(values[14])
		customer_inn := toString(values[15])
		customer_kpp := toString(values[16])
		customer_address := toString(values[17])
		consignee_name := toString(values[18])

		var on_order_coupon, on_order_coupon_fixed float64
		if values[19] != nil {
			s := values[19].(pgtype.Numeric)
			s.AssignTo(&on_order_coupon)
		}

		if values[20] != nil {
			s := values[20].(pgtype.Numeric)
			s.AssignTo(&on_order_coupon_fixed)
		}

		contractor_number := toString(values[21])
		if contractor_number != "" && len(contractor_number) < 10 {
			s := "00000000000"
			s = s + contractor_number
			contractor_number = s[len(s)-11:]
		}

		entry := OrderDetails{
			id,
			contractor_number,
			sum,
			sum_with_tax,
			status,
			toTime(values[3]),
			toTime(values[4]),
			toTime(values[5]),
			seller_id,
			seller_name,
			seller_inn,
			seller_kpp,
			seller_address,
			buyer_id,
			buyer,
			customer_id,
			customer_name,
			customer_inn,
			customer_kpp,
			customer_address,
			consignee_name,
			on_order_coupon,
			on_order_coupon_fixed,
			toTime(values[22]),
			toTime(values[23]),
			toTime(values[24]),
			toString(values[26]),
			toString(values[27]),
		}
		orders = append(orders, entry)
	}

	return orders, rows.Err()
}

type OrderLines []OrderLine

type OrderLine struct {
	ProductId        int     `json:"product_id"`
	Name             string  `json:"name"`
	Code             string  `json:"code"`
	Warehouse        string  `json:"warehouse"`
	WarehouseAddress string  `json:"warehouse_address"`
	Count            float64 `json:"count"`
	Price            float64 `json:"price"`
	Nds              float64 `json:"nds"`
	CouponPercent    float64 `json:"coupon_percent"`
	CouponFixed      float64 `json:"coupon_fixed"`
	CouponValue      float64 `json:"coupon_value"`
	Comment          string  `json:"comment"`
	Sum              float64 `json:"sum"`
	Tax              float64 `json:"tax"`
	SumWithTax       float64 `json:"sum_with_tax"`
}

func (db *ProdDBHelper) getOrder(ctx context.Context, userInfo UserInfo, orderId int) (OrderLines, error) {

	filterUsers, args := db.ordersAccessRightsFilter(userInfo)

	queryOrderDetails := `
		SELECT 
			pp.id,
			pp.name,
			pp.code,
			sw.name AS warehouse,
			oi.count, 
			oi.item_price, 
			oi.rate_nds, 
			oi.coupon_percent, 
			oi.coupon_fixed, 
			oi.coupon_value, 
			oi.comment,
			round((oi.count * ((((oi.item_price - oi.coupon_fixed) * ((100)::numeric - oi.coupon_percent)) / (100)::numeric))::double precision)::numeric, 2) AS sum,
			sw.address AS warehouse_address
		FROM order_orderitem oi
			JOIN order_order oo ON (oi.order_id = oo.id)
			JOIN product_modification pm ON (oi.modification_id = pm.id)
			JOIN product_product pp ON (pm.product_id = pp.id)
			LEFT JOIN supplier_warehouse sw ON (sw.id = oi.warehouse_id)
		WHERE oo.id=$`

	args = append(args, orderId)
	queryOrderDetails += strconv.Itoa(len(args))

	queryOrderDetails = queryOrderDetails + filterUsers + `	ORDER BY oi.id DESC`
	rows, _ := db.pool.Query(ctx, queryOrderDetails, args...)

	orderDetails := make([]OrderLine, 0)
	for rows.Next() {

		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		id := int(values[0].(int32))
		name := toString(values[1])
		code := toString(values[2])
		warehouse := toString(values[3])
		count := values[4].(float64)
		var price float64
		if values[5] != nil {
			s := values[5].(pgtype.Numeric)
			s.AssignTo(&price)
		}
		nds := toFloat(values[6])
		var coupon_percent float64
		if values[7] != nil {
			s := values[7].(pgtype.Numeric)
			s.AssignTo(&coupon_percent)
		}
		var coupon_fixed float64
		if values[8] != nil {
			s := values[8].(pgtype.Numeric)
			s.AssignTo(&coupon_fixed)
		}
		var coupon_value float64
		if values[9] != nil {
			s := values[9].(pgtype.Numeric)
			s.AssignTo(&coupon_value)
		}
		comment := toString(values[10])
		var sum float64
		if values[11] != nil {
			s := values[11].(pgtype.Numeric)
			s.AssignTo(&sum)
		}

		entry := OrderLine{
			id,
			name,
			code,
			warehouse,
			toString(values[12]),
			count,
			price,
			nds,
			coupon_percent,
			coupon_fixed,
			coupon_value,
			comment,
			sum,
			math.Round(sum*nds) / 100,
			sum + math.Round(sum*nds)/100,
		}
		orderDetails = append(orderDetails, entry)
	}

	return orderDetails, rows.Err()
}

type CartNumbers struct {
	OrdersCount int     `json:"ordersCount"`
	TotalSum    float64 `json:"totalSum"`
	ItemsCount  int     `json:"itemsCount"`
}

func (db *ProdDBHelper) getCartNumbers(ctx context.Context, ui UserInfo) (*CartNumbers, error) {
	cn := CartNumbers{}
	err := db.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT oo.id) AS orders_count, COALESCE(SUM(items_discounted_price), 0) as total_sum, COUNT(oi.id) AS items_count FROM order_order oo
			JOIN (
				SELECT
					oi.id,
					oi.order_id,
					pp.id product_id,
					pp.name,
					pp.code,
					pc.name as category,
					sw.name AS warehouse,
					sw.id AS warehouse_id,
					oi.count,
					oi.item_price,
					oi.rate_nds,
					oi.coupon_percent,
					oi.coupon_fixed,
					oi.coupon_value,
					oi.comment,
					round((((((oi.item_price - oi.coupon_fixed) * ((100)::numeric - oi.coupon_percent)) / (100)::numeric))::double precision)::numeric, 2)*count AS items_discounted_price

				FROM order_orderitem oi
						JOIN order_order oo ON (oi.order_id = oo.id)
						JOIN product_modification pm ON (oi.modification_id = pm.id)
						JOIN product_product pp ON (pm.product_id = pp.id)
						LEFT JOIN supplier_warehouse sw ON (sw.id = oi.warehouse_id)
						LEFT JOIN product_category pc ON (pc.id = category_id)
			) oi ON oo.id = oi.order_id
		WHERE user_id=$1 AND status_id=18 AND deleted != TRUE
		`, ui.Id).Scan(&cn.OrdersCount, &cn.TotalSum, &cn.ItemsCount)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve user cart numbers: %v", err)
	}

	return &cn, nil

}

func (db *ProdDBHelper) getCompareItemsCount(ctx context.Context, ui UserInfo) (int, error) {
	var ci int
	err := db.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM compare_comparelist c JOIN compare_compareitem ci ON c.id = ci.compare_id WHERE c.name=$1
		`, ui.CompareList).Scan(&ci)
	if err != nil {
		return 0, fmt.Errorf("Failed to retrieve user compare items count: %v", err)
	}

	return ci, nil
}

type CartItem struct {
	OrderId     int     `json:"orderId"`
	ProductId   int     `json:"productId"`
	ProductCode string  `json:"productCode"`
	Count       float64 `json:"count"`
	ProductName string  `json:"productName"`
	ItemPrice   float64 `json:"itemPrice"`
}

func (db *ProdDBHelper) getCartItems(ctx context.Context, ui UserInfo) (map[int]CartItem, error) {
	rows, _ := db.pool.Query(ctx, `
		SELECT oo.id, product_id, oi.code, count, oi.name, item_discounted_price FROM order_order oo
			JOIN (
				SELECT
					oi.id,
					oi.order_id,
					pp.id product_id,
					pp.name,
					pp.code,
					pc.name as category,
					sw.name AS warehouse,
					sw.id AS warehouse_id,
					oi.count,
					oi.item_price,
					oi.rate_nds,
					oi.coupon_percent,
					oi.coupon_fixed,
					oi.coupon_value,
					oi.comment,
					round((((((oi.item_price - oi.coupon_fixed) * ((100)::numeric - oi.coupon_percent)) / (100)::numeric))::double precision)::numeric, 2)*count AS item_discounted_price

				FROM order_orderitem oi
						JOIN order_order oo ON (oi.order_id = oo.id)
						JOIN product_modification pm ON (oi.modification_id = pm.id)
						JOIN product_product pp ON (pm.product_id = pp.id)
						LEFT JOIN supplier_warehouse sw ON (sw.id = oi.warehouse_id)
						LEFT JOIN product_category pc ON (pc.id = category_id)
			) oi ON oo.id = oi.order_id
		WHERE user_id=$1 AND status_id=18 AND deleted != TRUE
		
		`, ui.Id)

	cartItems := make(map[int]CartItem, 0)
	for rows.Next() {

		var ci CartItem
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		ci.OrderId = toInt(values[0])
		ci.ProductId = toInt(values[1])
		ci.ProductCode = toString(values[2])
		ci.Count = toFloat(values[3])
		ci.ProductName = toString(values[4])
		ci.ItemPrice = toFloat(values[5])

		cartItems[ci.ProductId] = ci
	}

	return cartItems, nil
}
