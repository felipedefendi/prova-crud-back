package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

const (
	DB_USER     = "postgres"
	DB_PASSWORD = "3163"
	DB_NAME     = "prova_crud"
	DB_PORT     = "5433"
)

var db *sql.DB

type Imovel struct {
	ID         int       `json:"id"`
	Descricao  string    `json:"descricao"`
	DataCompra time.Time `json:"dataCompra"`
	Endereco   string    `json:"endereco"`
	Comodos    []Comodo  `json:"comodos"`
}

type Comodo struct {
	ID       int    `json:"id"`
	Nome     string `json:"nome"`
	ImovelID int    `json:"imovel_id"`
}

func initDB() {
	psqlInfo := fmt.Sprintf("host=localhost port=%s user=%s password=%s dbname=%s sslmode=disable",
		DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully connected to database!")
}

func createImovel(c *gin.Context) {
	type InputImovel struct {
		Descricao  string `json:"descricao"`
		DataCompra string `json:"dataCompra"`
		Endereco   string `json:"endereco"`
	}

	var input InputImovel
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("Erro ao fazer bind do JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parsedDate, err := time.Parse("2006-01-02", input.DataCompra)
	if err != nil {
		log.Printf("Erro ao converter a data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de data inválido. Use yyyy-MM-dd"})
		return
	}

	sqlStatement := `INSERT INTO imoveis (descricao, data_compra, endereco) VALUES ($1, $2, $3) RETURNING id`
	var id int
	err = db.QueryRow(sqlStatement, input.Descricao, parsedDate, input.Endereco).Scan(&id)
	if err != nil {
		log.Printf("Erro ao inserir no banco de dados: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Imóvel criado com sucesso!", "id": id})
}

func getImoveis(c *gin.Context) {
	var imoveis []Imovel
	rows, err := db.Query("SELECT id, descricao, data_compra, endereco FROM imoveis")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar imóveis"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var imovel Imovel
		err := rows.Scan(&imovel.ID, &imovel.Descricao, &imovel.DataCompra, &imovel.Endereco)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao ler os dados"})
			return
		}

		imovel.Comodos, err = getComodosByImovel(imovel.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar cômodos"})
			return
		}

		imoveis = append(imoveis, imovel)
	}

	c.JSON(http.StatusOK, imoveis)
}

func getComodosByImovel(imovelID int) ([]Comodo, error) {
	rows, err := db.Query("SELECT id, nome FROM comodos WHERE imovel_id = $1", imovelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comodos []Comodo
	for rows.Next() {
		var comodo Comodo
		err := rows.Scan(&comodo.ID, &comodo.Nome)
		if err != nil {
			return nil, err
		}
		comodos = append(comodos, comodo)
	}
	return comodos, nil
}

func updateImovel(c *gin.Context) {
	id := c.Param("id")

	type InputImovel struct {
		Descricao  string `json:"descricao"`
		DataCompra string `json:"dataCompra"`
		Endereco   string `json:"endereco"`
	}

	var input InputImovel
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("Erro ao fazer bind do JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var parsedDate time.Time
	var err error
	if len(input.DataCompra) > 10 {
		parsedDate, err = time.Parse(time.RFC3339, input.DataCompra)
	} else {
		parsedDate, err = time.Parse("2006-01-02", input.DataCompra)
	}

	if err != nil {
		log.Printf("Erro ao converter a data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de data inválido. Use yyyy-MM-dd ou yyyy-MM-ddTHH:MM:SSZ"})
		return
	}

	sqlStatement := `UPDATE imoveis SET descricao=$2, data_compra=$3, endereco=$4 WHERE id=$1`
	_, err = db.Exec(sqlStatement, id, input.Descricao, parsedDate, input.Endereco)
	if err != nil {
		log.Printf("Erro ao atualizar o banco de dados: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Imóvel atualizado com sucesso!"})
}

func deleteImovel(c *gin.Context) {
	id := c.Param("id")

	sqlStatement := `DELETE FROM imoveis WHERE id=$1`
	_, err := db.Exec(sqlStatement, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Imóvel deletado com sucesso!"})
}

func createComodo(c *gin.Context) {
	var comodo Comodo
	if err := c.ShouldBindJSON(&comodo); err != nil {
		log.Printf("Erro ao fazer bind do JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sqlStatement := `INSERT INTO comodos (nome, imovel_id) VALUES ($1, $2) RETURNING id`
	err := db.QueryRow(sqlStatement, comodo.Nome, comodo.ImovelID).Scan(&comodo.ID)
	if err != nil {
		log.Printf("Erro ao inserir no banco de dados: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, comodo)
}

func deleteComodo(c *gin.Context) {
	id := c.Param("id")

	sqlStatement := `DELETE FROM comodos WHERE id=$1`
	_, err := db.Exec(sqlStatement, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Cômodo deletado com sucesso!"})
}

func main() {
	initDB()
	defer db.Close()

	router := gin.Default()
	router.Use(cors.Default())

	router.POST("/imoveis", createImovel)
	router.GET("/imoveis", getImoveis)
	router.PUT("/imoveis/:id", updateImovel)
	router.DELETE("/imoveis/:id", deleteImovel)

	router.POST("/comodos", createComodo)
	router.DELETE("/comodos/:id", deleteComodo)

	router.Run(":8080")
}
