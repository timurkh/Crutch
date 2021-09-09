package main

import (
	"context"
	"fmt"
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
}

type OrdersFilter struct {
	Start            time.Time `schema:"start"`
	End              time.Time `schema:"end"`
	Text             string    `schema:"text"`
	SelectedStatuses []int     `schema:"selectedStatuses[]"`
}

type DBHelper struct {
	pool *pgxpool.Pool
}

func initDBHelper(host string, user string, password string, database string) (*DBHelper, error) {
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

	db := DBHelper{pool}
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
}

func (db *DBHelper) getUserInfo(userId int) (*UserDBInfo, error) {
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
			COALESCE( contractor_id, 0) contractor_id
		FROM 
			core_user cu 
			LEFT JOIN (SELECT user_id, TRUE as can_read_orders FROM core_user_user_permissions up WHERE permission_id=1067) ro ON (ro.user_id=cu.id AND cu.is_staff) 
			LEFT JOIN (SELECT user_id, TRUE as can_read_buyers FROM core_user_user_permissions up WHERE permission_id=286) rb ON (rb.user_id=cu.id AND cu.is_staff) 
			LEFT JOIN (SELECT user_id, TRUE as can_read_sellers FROM core_user_user_permissions up WHERE permission_id=678) rs ON (rs.user_id=cu.id AND cu.is_staff) 
			LEFT JOIN core_user_contractors cuc ON cu.id = cuc.user_id
		WHERE cu.id =$1`, userId).Scan(&ui.first_name, &ui.last_name, &ui.email, &ui.is_superuser, &ui.is_staff, &ui.is_company_admin, &ui.can_read_orders, &ui.can_read_buyers, &ui.can_read_sellers, &ui.verified, &ui.blocked, &ui.contractor_id)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve user info: %v", err)
	}

	return &ui, nil

}

func (db *DBHelper) getUserConsigneeCities(ctx context.Context, userInfo UserInfo) (cities []City, err error) {

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

func (db *DBHelper) getProductEntries(ctx context.Context, product_ids []int, products_score map[int]float64, userInfo UserInfo, city_id int, inStockOnly bool, supplier string) (products []map[string]interface{}, err error) {

	args := []interface{}{product_ids}

	supplier_warehouses := ""
	if userInfo.Admin {
		if city_id > 0 {
			supplier_warehouses = `
			AND pr.warehouse_id IN (
			SELECT sw.id 
			FROM supplier_warehouse sw 
				INNER JOIN supplier_warehouse_delivery_cities swc ON (sw.id = swc.warehouse_id) 
			WHERE sw.is_visible = true  
				AND swc.city_id = $2)`
			args = append(args, city_id)
		}
	} else {
		client_cities := `
		SELECT DISTINCT city_id 
		FROM core_user_contractors cuc 
			JOIN consignee_consignee con USING(contractor_id) 
			JOIN company_city com on com.id = con.city_id 
		WHERE cuc.user_id=$2`

		args = append(args, userInfo.Id)

		if city_id > 0 {
			client_cities = client_cities + " AND city_id = $3"
			args = append(args, city_id)
		}

		supplier_warehouses = `
		AND pr.warehouse_id IN (
		SELECT sw.id 
		FROM supplier_warehouse sw 
			INNER JOIN supplier_warehouse_delivery_cities swc ON (sw.id = swc.warehouse_id) 
		WHERE sw.is_visible = true  
			AND swc.city_id IN (` + client_cities + `))`
	}

	product_quantity := `
	SELECT
		pp.id,
		SUM(pr.rest) AS rest,
		ordering
	FROM
		product_product pp
		JOIN product_modification pm ON ( pp.id = pm.product_id )
		JOIN product_rest  pr ON ( pm.id = pr.modification_id )
		JOIN (SELECT * FROM unnest($1::int[]) WITH ORDINALITY) x (id, ordering) ON (pp.id = x.id)
	WHERE
		pp.deleted = false
		AND pp.is_reference = false
		AND pp.b_placement_state = 'placed'
		AND pp.category_id IS NOT NULL
		AND pp.hidden = false  
	` + supplier_warehouses + `
	GROUP BY pp.id, ordering`

	query := `
	SELECT
		pp.id,
		pc.name as category_name,
		pp.name,
		pp.code,
		pp.description,
		pr.rest,
		pp.product_price,
		cc.name as supplier
	FROM 
		( ` + product_quantity + ` 
			) pr 
		JOIN product_product pp USING (id) 
		JOIN product_category pc ON ( pp.category_id = pc.id )
		JOIN product_suppliercategory psc ON (pp.supplier_category_id = psc.id)
		JOIN company_company cc ON (cc.object_id=supplier_id AND content_type_id=186)
	WHERE 
		pc.hidden = false
		AND (NOT pr.rest = 0.0`

	if !inStockOnly {
		query += " OR pp.enable_preorder = true"
	}
	query += ")"

	if supplier != "" {
		args = append(args, "%"+supplier+"%")
		query += " AND cc.name ILIKE $" + strconv.Itoa(len(args)) + ""
	}

	query += " ORDER BY ordering"

	rows, _ := db.pool.Query(ctx, query, args...)

	products = make([]map[string]interface{}, 0)
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

		entry := map[string]interface{}{
			"id":          id,
			"category":    category,
			"code":        code,
			"name":        name,
			"description": description,
			"rest":        toFloat(values[5]),
			"price":       price,
			"supplier":    supplier,
		}
		entry["score"] = products_score[id]
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

func toDate(v interface{}) string {
	if v != nil {
		return v.(time.Time).Format("2006-01-02")
	}
	return ""
}

func toTime(v interface{}) string {
	if v != nil {
		return v.(time.Time).Format("15:04:05")
	}
	return ""
}

func (db *DBHelper) ordersAccessRightsFilter(userInfo UserInfo, args []interface{}) (string, []interface{}) {
	filterUsers := ""
	if !userInfo.Admin && !(userInfo.Staff && userInfo.CanReadOrders) {

		args = append(args, userInfo.Id)
		if userInfo.CompanyAdmin {
			filterUsers = ` AND oo.contractor_id in (
				SELECT contractor_id FROM core_user_contractors 
				WHERE user_id=$`
		} else {

			filterUsers = ` AND oo.user_id in (
				SELECT ou.id 
				FROM core_user ou 
					JOIN core_user cu ON (ou.lft <= cu.rght  AND ou.lft >= cu.lft  AND ou.tree_id = cu.tree_id) 
				WHERE cu.id=$`
		}
		filterUsers += strconv.Itoa(len(args)) + ")"
	}
	return filterUsers, args
}

func (db *DBHelper) getCounterparts(ctx context.Context, userInfo UserInfo, filter CounterpartsFilter) (counterparts []map[string]interface{}, err error) {

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
				CASE WHEN content_type_id=79 THEN contractor_date_joined 
					WHEN content_type_id=186 THEN supplier_date_joined
					ELSE NULL 
				END AS date_joined,
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
				LEFT JOIN (SELECT COUNT(*) supplier_user_count, MIN(date_joined) AS supplier_date_joined, MAX(last_login) AS supplier_last_login, supplier_id 
					FROM core_user WHERE verified=TRUE AND blocked=FALSE GROUP BY supplier_id) cu 
					ON (object_id = cu.supplier_id AND content_type_id=186) 
				LEFT JOIN (SELECT COUNT (*) contractor_user_count, MIN(date_joined) contractor_date_joined, MAX(last_login) contractor_last_login, contractor_id 
					FROM core_user_contractors JOIN core_user cu ON (cu.id = user_id AND verified=TRUE and blocked=FALSE) GROUP BY contractor_id) cuc 
					ON (object_id=cuc.contractor_id AND content_type_id=79)
				LEFT JOIN (SELECT COUNT (*) supplier_order_count, supplier_id FROM order_order WHERE id NOT IN (1, 17) `
	query += dateFilter
	query += ` GROUP BY supplier_id) so
					ON (object_id = so.supplier_id AND content_type_id=186)
				LEFT JOIN (SELECT COUNT (*) contractor_order_count, contractor_id FROM order_order WHERE id NOT IN (1, 17) `
	query += dateFilter
	query += ` GROUP BY contractor_id) co
					ON (object_id = co.contractor_id AND content_type_id=79)
				LEFT JOIN supplier_supplierprofile ss ON cc.object_id = ss.supplier_id and content_type_id=186
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
		)s WHERE user_count>0 `

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
			"date_joined":    values[8],
			"last_login":     values[9],
			"order_count":    values[10],
			"ogrn":           toString(values[11]),
			"actual_address": values[12],
			"director_name":  toString(values[13]),
			"contact_name":   toString(values[14]),
			"phone":          toString(values[15]),
			"bank":           toString(values[16]),
			"bik":            toString(values[17]),
			"corr_account":   toString(values[18]),
			"pay_account":    toString(values[19]),
			"extra_data":     toString(values[20]),
			"account":        toString(values[21]),
			"bank_phone":     toString(values[22]),
			"IBAN":           toString(values[23]),
			"SWIFT":          toString(values[24]),
			"country":        toString(values[25]),
			"city":           toString(values[26]),
			"seller_email":   toString(values[27]),
			"seller_site":    toString(values[28]),
			"seller_phone":   toString(values[29]),
			"admins":         values[30],
		}

		counterparts = append(counterparts, entry)
	}

	return counterparts, rows.Err()
}

func (db *DBHelper) getOrders(ctx context.Context, userInfo UserInfo, ordersFilter OrdersFilter) (orders []map[string]interface{}, err error) {

	args := make([]interface{}, 0)
	filterUsers, args := db.ordersAccessRightsFilter(userInfo, args)

	queryOrders := `
		SELECT oo.id, 
			ov.order_sum,
			os.status,
			dor.date_ordered,
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
			da.date_accepted
		FROM order_order oo 
			JOIN (
				SELECT oo.id, 
					round(((((sum((oi.count * ((((oi.item_price - oi.coupon_fixed) * ((100)::numeric - oi.coupon_percent)) / (100)::numeric))::double precision)) * (((100)::numeric - oo.on_order_coupon))::double precision) / (100)::double precision) - (oo.on_order_coupon_fixed)::double precision))::numeric, 2) AS order_sum
				FROM order_order oo 
					LEFT JOIN order_orderitem oi ON (oo.id = oi.order_id)
				GROUP BY oo.id
			) ov USING (id)
			JOIN (
				SELECT object_id_int AS order_id, MIN(rr.date_created) AS date_ordered
				FROM reversion_version rv JOIN reversion_revision rr ON rv.revision_id = rr.id 
				WHERE content_type_id=115 and serialized_data::json->0->'fields'->>'status' = '1'
				GROUP BY object_id_int) dor ON dor.order_id = oo.id 
			LEFT JOIN (
				SELECT object_id_int AS order_id, MIN(rr.date_created) AS date_shipped
				FROM reversion_version rv JOIN reversion_revision rr ON rv.revision_id = rr.id 
				WHERE content_type_id=115 and serialized_data::json->0->'fields'->>'status' = '21'
				GROUP BY object_id_int) ds ON ds.order_id = oo.id 
			LEFT JOIN (
				SELECT object_id_int AS order_id, MIN(rr.date_created) AS date_delivered
				FROM reversion_version rv JOIN reversion_revision rr ON rv.revision_id = rr.id 
				WHERE content_type_id=115 and serialized_data::json->0->'fields'->>'status' = '15'
				GROUP BY object_id_int) dd ON dd.order_id = oo.id 
			LEFT JOIN (
				SELECT object_id_int AS order_id, MIN(rr.date_created) AS date_accepted
				FROM reversion_version rv JOIN reversion_revision rr ON rv.revision_id = rr.id 
				WHERE content_type_id=115 and serialized_data::json->0->'fields'->>'status' = '22'
				GROUP BY object_id_int) da ON da.order_id = oo.id 
			JOIN order_orderstatus os ON (oo.status_id = os.id)
			JOIN company_company seller ON (seller.object_id=oo.supplier_id AND seller.content_type_id=186)
			JOIN core_user cu ON (cu.id = oo.user_id)
			LEFT JOIN consignee_consignee cc ON (cc.id = oo.consignee_id)
			JOIN company_company customer ON (customer.object_id=oo.contractor_id AND customer.content_type_id=79)
		WHERE oo.status_id NOT IN (17, 18)`

	// filter by date
	args = append(args, ordersFilter.End)
	queryOrders += " AND dor.date_ordered<$" + strconv.Itoa(len(args))

	args = append(args, ordersFilter.Start)
	queryOrders += " AND dor.date_ordered>$" + strconv.Itoa(len(args))

	// filter by text
	if ordersFilter.Text != "" {
		args = append(args, "%"+ordersFilter.Text+"%")
		queryOrders += " AND (seller.name ILIKE $" + strconv.Itoa(len(args))
		queryOrders += " OR customer.name ILIKE $" + strconv.Itoa(len(args))
		queryOrders += " OR cc.name ILIKE $" + strconv.Itoa(len(args))
		queryOrders += " OR cu.last_name ILIKE $" + strconv.Itoa(len(args))
		queryOrders += " OR cu.first_name ILIKE $" + strconv.Itoa(len(args))
		queryOrders += " OR cu.middle_name ILIKE $" + strconv.Itoa(len(args)) + ")"
	}

	if len(ordersFilter.SelectedStatuses) > 0 {
		args = append(args, ordersFilter.SelectedStatuses)
		queryOrders += " AND status_id = ANY($" + strconv.Itoa(len(args)) + ")"
	}

	queryOrders = queryOrders + filterUsers + `	ORDER BY dor.date_ordered DESC`
	rows, _ := db.pool.Query(ctx, queryOrders, args...)

	orders = make([]map[string]interface{}, 0)
	for rows.Next() {

		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		id := int(values[0].(int32))
		var sum float64
		if values[1] != nil {
			s := values[1].(pgtype.Numeric)
			s.AssignTo(&sum)
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

		s := "000000000"
		s = s + toString(values[21])
		contractor_number := s[len(s)-10:]

		entry := map[string]interface{}{
			"id":                    id,
			"contractor_number":     contractor_number,
			"sum":                   sum,
			"status":                status,
			"ordered_date":          values[3],
			"closed_date":           values[4],
			"shipping_date_est":     values[5],
			"seller_id":             seller_id,
			"seller_name":           seller_name,
			"seller_inn":            seller_inn,
			"seller_kpp":            seller_kpp,
			"seller_address":        seller_address,
			"buyer_id":              buyer_id,
			"buyer":                 buyer,
			"customer_id":           customer_id,
			"customer_name":         customer_name,
			"customer_inn":          customer_inn,
			"customer_kpp":          customer_kpp,
			"customer_address":      customer_address,
			"consignee_name":        consignee_name,
			"on_order_coupon":       on_order_coupon,
			"on_order_coupon_fixed": on_order_coupon_fixed,
			"shipped_date":          values[22],
			"delivered_date":        values[23],
			"accepted_date":         values[24],
		}
		orders = append(orders, entry)
	}

	return orders, rows.Err()
}

func (db *DBHelper) getOrder(ctx context.Context, userInfo UserInfo, orderId int) (orders []map[string]interface{}, err error) {

	args := []interface{}{orderId}
	filterUsers, args := db.ordersAccessRightsFilter(userInfo, args)

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
			round((oi.count * ((((oi.item_price - oi.coupon_fixed) * ((100)::numeric - oi.coupon_percent)) / (100)::numeric))::double precision)::numeric, 2) AS sum
		FROM order_orderitem oi
			JOIN order_order oo ON (oi.order_id = oo.id)
			JOIN product_modification pm ON (oi.modification_id = pm.id)
			JOIN product_product pp ON (pm.product_id = pp.id)
			LEFT JOIN supplier_warehouse sw ON (sw.id = oi.warehouse_id)
		WHERE oo.id=$1`
	queryOrderDetails = queryOrderDetails + filterUsers + `	ORDER BY oi.id DESC`
	rows, _ := db.pool.Query(ctx, queryOrderDetails, args...)

	orderDetails := make([]map[string]interface{}, 0)
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

		entry := map[string]interface{}{
			"product_id":     id,
			"name":           name,
			"code":           code,
			"warehouse":      warehouse,
			"count":          count,
			"price":          price,
			"nds":            nds,
			"coupon_percent": coupon_percent,
			"coupon_fixed":   coupon_fixed,
			"coupon_value":   coupon_value,
			"comment":        comment,
			"sum":            sum,
		}
		orderDetails = append(orderDetails, entry)
	}

	return orderDetails, rows.Err()
}
