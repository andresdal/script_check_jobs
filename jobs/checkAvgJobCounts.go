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
	
	// PostgreSQL connection string
	db_user := os.Getenv("PG_DB_USERNAME")
	db_name := os.Getenv("PG_DB_DATABASE")
	db_password := os.Getenv("PG_DB_PASSWORD")
	db_host := os.Getenv("PG_DB_HOST")
	db_port := os.Getenv("PG_DB_PORT")

	connStr := "user=" + db_user + " dbname=" + db_name + " password=" + db_password + " host=" + db_host + " port=" + db_port + " sslmode=disable"

	// Open a connection to the database
	db, err := sql.Open("postgres", connStr)
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
	var porcentaje_limite_aceptable float64 = 90.0

	// Execute the query and scan the result into the count variable
	err = db.QueryRow(query).Scan(&porcentaje_variacion)
	if err != nil {
		log.Fatal("Failed to execute query: ", err)
	}

	genericErrorMessage := "CheckAvgDbCounts error:"
	errorMessage := fmt.Sprintf("El porcentaje de variación de clicks (%2.f) es menor al límite aceptable (%2.f).", porcentaje_variacion, porcentaje_limite_aceptable)

	if(porcentaje_variacion < porcentaje_limite_aceptable) {
		result, _ := redis_client.Get("check_avg_db_counts").Result()
		if result != "error" {
			redis_client.Set("check_avg_db_counts", "error", 0)
			slack_utils.SendMessage(genericErrorMessage + "\n" + errorMessage, channelID)
		}
		return
	}

	// if there is no error
	result, _ := redis_client.Get("check_avg_db_counts").Result()
	if result == "error" {
		redis_client.Set("check_avg_db_counts", "solved", 0)
	}
}