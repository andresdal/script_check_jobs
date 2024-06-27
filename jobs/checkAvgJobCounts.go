package jobs

import (
	"database/sql"
	"fmt"
	"log"
	"neptune/check_jobs/slack_utils"
	"os"

	"github.com/go-redis/redis"
	_ "github.com/lib/pq"
)

func CheckAvgDbCounts(channelID string) {
	redis_client := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379", // Dirección del servidor Redis
        Password: "",               // Contraseña, si no tienes una, déjala vacía
        DB:       0,                // Base de datos a usar
    })
	
	// PostgreSQL HIREABLE connection
	db_user_hir := os.Getenv("POSTGRES_USER_HIR")
	db_name_hir := os.Getenv("POSTGRES_DATABASE_HIR")
	db_password_hir := os.Getenv("POSTGRES_PASSWORD_HIR")
	db_host_hir := os.Getenv("POSTGRES_HOST_HIR")
	db_port_hir := os.Getenv("POSTGRES_PORT_HIR")

	connStrHir := "user=" + db_user_hir + " dbname=" + db_name_hir + " password=" + db_password_hir + " host=" + db_host_hir + " port=" + db_port_hir + " sslmode=disable"

	// Open a connection to the database
	db, err := sql.Open("postgres", connStrHir)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Prepare the SQL query string
	query := 	`WITH tiempo_actual AS (
					SELECT current_time AS hora_fin,
						(current_time - interval '1 hour') AS hora_inicio
				),
				clicks_ultimos_5_dias AS (
					SELECT AVG(daily_clicks) AS promedio_clicks_diarios
					FROM (
						SELECT DATE(ts) AS fecha, COUNT(*) AS daily_clicks
						FROM clickers, tiempo_actual
						WHERE ts::time BETWEEN tiempo_actual.hora_inicio AND tiempo_actual.hora_fin
						AND DATE(ts) BETWEEN current_date - interval '5 days' AND current_date - interval '1 day'
						GROUP BY DATE(ts)
					) AS subquery
				),
				clicks_hoy AS (
					SELECT COUNT(*) AS clicks_hoy
					FROM clickers, tiempo_actual
					WHERE ts::time BETWEEN tiempo_actual.hora_inicio AND tiempo_actual.hora_fin
					AND DATE(ts) = current_date
				)
				SELECT 
					(clicks_hoy.clicks_hoy  / clicks_ultimos_5_dias.promedio_clicks_diarios * 100) AS porcentaje_variacion
				FROM clicks_hoy, clicks_ultimos_5_dias;
				`

	// Variable to store the query result
	var porcentaje_variacion float64 
	var porcentaje_limite_aceptable float64 = 30.0

	// Execute the query and scan the result into the count variable
	err = db.QueryRow(query).Scan(&porcentaje_variacion)
	if err != nil {
		log.Fatal("Failed to execute query: ", err)
	}

	genericErrorMessage := "CheckAvgDbCounts error:"
	errorMessage := fmt.Sprintf("El porcentaje de variación de clicks (%.2f) es menor al límite aceptable (%.2f).", porcentaje_variacion, porcentaje_limite_aceptable)

	if(porcentaje_variacion < porcentaje_limite_aceptable) { // error
		result, _ := redis_client.Get("check_avg_db_counts_hir").Result()
		if result != "error" {
			redis_client.Set("check_avg_db_counts_hir", "error", 0)
			slack_utils.SendMessage("HIREABLE " + genericErrorMessage + "\n" + errorMessage, channelID)
		}
	} else { // ok
		result, _ := redis_client.Get("check_avg_db_counts_hir").Result()
		if result == "error" {
			redis_client.Set("check_avg_db_counts_hir", "solved", 0)
			slack_utils.SendMessage("Hireable CheckAvgDbCounts error SOLVED", channelID)
		}
	}

	// PostgreSQL WALLA connection
	db_user_wal := os.Getenv("POSTGRES_USER_WALLA")
	db_name_wal := os.Getenv("POSTGRES_DATABASE_WALLA")
	db_password_wal := os.Getenv("POSTGRES_PASSWORD_WALLA")
	db_host_wal := os.Getenv("POSTGRES_HOST_WALLA")
	db_port_wal := os.Getenv("POSTGRES_PORT_WALLA")

	connStrWalla := fmt.Sprintf("user=%s dbname=%s password=%s host=%s port=%s sslmode=disable",
    db_user_wal, db_name_wal, db_password_wal, db_host_wal, db_port_wal)

	// Open a connection to the database
	db, err = sql.Open("postgres", connStrWalla)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Execute the query and scan the result into the count variable
	err = db.QueryRow(query).Scan(&porcentaje_variacion)
	if err != nil {
		log.Fatal("Failed to execute query: ", err)
	}

	errorMessage = fmt.Sprintf("El porcentaje de variación de clicks (%.2f) es menor al límite aceptable (%.2f).", porcentaje_variacion, porcentaje_limite_aceptable)

	if(porcentaje_variacion < porcentaje_limite_aceptable) { // error
		result, _ := redis_client.Get("check_avg_db_counts_walla").Result()
		if result != "error" {
			redis_client.Set("check_avg_db_counts_walla", "error", 0)
			slack_utils.SendMessage("WALLA " + genericErrorMessage + "\n" + errorMessage, channelID)
		}
	} else { // ok
		result, _ := redis_client.Get("check_avg_db_counts_walla").Result()
		if result == "error" {
			redis_client.Set("check_avg_db_counts_walla", "solved", 0)
			slack_utils.SendMessage("Walla CheckAvgDbCounts error SOLVED", channelID)
		}
	}
}