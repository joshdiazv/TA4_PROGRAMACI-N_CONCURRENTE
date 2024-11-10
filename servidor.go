package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Movie struct {
	MovieID int
	Title   string
	Genres  []string // Lista de géneros
}

type Rating struct {
	UserID  int
	MovieID int
	Rating  float64
}

type Recommendation struct {
	MovieID   int
	Title     string
	Genres    []string
	AvgRating float64
	Count     int
}

var movies = make(map[int]Movie)
var ratings = make([]Rating, 0)

// Mapa para almacenar las recomendaciones de diferentes clientes y combinarlas
var genreRecommendations = make(map[string]map[int]*Recommendation)
var mutex sync.Mutex // Mutex para proteger el acceso concurrente al mapa

func main() {
	// Cargar películas y calificaciones
	loadMovies("Dataset/movies.csv")
	loadRatings("Dataset/ratings.csv")

	// Iniciar servidor
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error al configurar el servidor:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Servidor escuchando en puerto 8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error de conexión:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	fmt.Fprintln(writer, "Bienvenido al sistema de recomendación de películas!")
	writer.Flush()

	// Pedir ID de usuario
	fmt.Fprintln(writer, "Por favor, ingresa tu ID de usuario:")
	writer.Flush()
	var userID int
	fmt.Fscanln(conn, &userID)

	// Confirmación de que los archivos CSV se leyeron correctamente
	fmt.Fprintln(writer, "Los archivos CSV se leyeron correctamente. Puedes empezar a elegir un género para obtener recomendaciones.")
	writer.Flush()

	// Mostrar los géneros disponibles
	genres := getTopGenres()
	fmt.Fprintln(writer, "Por favor, selecciona uno de los siguientes géneros:")
	for i, genre := range genres {
		fmt.Fprintf(writer, "%d. %s\n", i+1, genre)
	}
	writer.Flush()

	// Indicar el fin de la lista de géneros
	fmt.Fprintln(writer, "[END_OF_GENRES]")
	writer.Flush()

	// Esperar la selección de género
	var genreIndex int
	fmt.Fscanln(conn, &genreIndex)

	// Validar el índice del género
	if genreIndex < 1 || genreIndex > len(genres) {
		fmt.Fprintln(writer, "Índice de género no válido.")
		writer.Flush()
		return
	}

	selectedGenre := genres[genreIndex-1]
	recommendations := getMoviesByGenre(selectedGenre)

	// Agregar y combinar recomendaciones en el mapa global
	combineRecommendations(selectedGenre, recommendations)
	displayCombinedRecommendations(writer, selectedGenre)
}

func combineRecommendations(genre string, recommendations []Movie) {
	mutex.Lock()
	defer mutex.Unlock()

	if _, exists := genreRecommendations[genre]; !exists {
		genreRecommendations[genre] = make(map[int]*Recommendation)
	}

	for _, movie := range recommendations {
		avgRating := calculateAverageRating(movie.MovieID)
		if rec, exists := genreRecommendations[genre][movie.MovieID]; exists {
			// Si ya existe una recomendación, actualizar el promedio de calificación
			rec.AvgRating = (rec.AvgRating*float64(rec.Count) + avgRating) / float64(rec.Count+1)
			rec.Count++
		} else {
			// Si es una nueva recomendación, agregarla al mapa
			genreRecommendations[genre][movie.MovieID] = &Recommendation{
				MovieID:   movie.MovieID,
				Title:     movie.Title,
				Genres:    movie.Genres,
				AvgRating: avgRating,
				Count:     1,
			}
		}
	}
}

func displayCombinedRecommendations(writer *bufio.Writer, genre string) {
	mutex.Lock()
	defer mutex.Unlock()

	if recs, exists := genreRecommendations[genre]; exists {
		fmt.Fprintln(writer, "Películas recomendadas combinadas para el género:", genre)
		count := 0
		for _, rec := range recs {
			if count >= 5 { // Limitar a las 5 primeras
				break
			}
			fmt.Fprintf(writer, "%d. Título: %s, Géneros: %s, Calificación Promedio Combinada: %.2f\n",
				count+1, rec.Title, strings.Join(rec.Genres, ", "), rec.AvgRating)
			count++
		}
	} else {
		fmt.Fprintln(writer, "No se encontraron recomendaciones para el género seleccionado.")
	}
	writer.Flush()
}

// Función para calcular la calificación promedio de una película
func calculateAverageRating(movieID int) float64 {
	var totalRating float64
	var count int
	for _, rating := range ratings {
		if rating.MovieID == movieID {
			totalRating += rating.Rating
			count++
		}
	}

	if count == 0 {
		return 0.0
	}
	return totalRating / float64(count)
}

func getTopGenres() []string {
	genreCount := make(map[string]int)
	for _, movie := range movies {
		for _, genre := range movie.Genres {
			genreCount[genre]++
		}
	}

	var genreList []string
	for genre := range genreCount {
		genreList = append(genreList, genre)
	}

	sort.Slice(genreList, func(i, j int) bool {
		return genreCount[genreList[i]] > genreCount[genreList[j]]
	})

	if len(genreList) > 15 {
		genreList = genreList[:15]
	}
	return genreList
}

func getMoviesByGenre(preferredGenre string) []Movie {
	var recommendedMovies []Movie
	for _, movie := range movies {
		for _, genre := range movie.Genres {
			if strings.Contains(strings.ToLower(genre), strings.ToLower(preferredGenre)) {
				recommendedMovies = append(recommendedMovies, movie)
				break
			}
		}
	}

	if len(recommendedMovies) > 5 {
		recommendedMovies = recommendedMovies[:5]
	}
	return recommendedMovies
}

// Cargar las películas desde un archivo CSV
func loadMovies(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error al abrir archivo de películas:", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	_, err = reader.Read()
	if err != nil {
		fmt.Println("Error al leer archivo de películas:", err)
		return
	}

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		movieID, _ := strconv.Atoi(record[0])
		genres := strings.Split(record[2], "|")

		movies[movieID] = Movie{
			MovieID: movieID,
			Title:   record[1],
			Genres:  genres,
		}
	}
}

// Cargar las calificaciones desde un archivo CSV
func loadRatings(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error al abrir archivo de calificaciones:", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	_, err = reader.Read()
	if err != nil {
		fmt.Println("Error al leer archivo de calificaciones:", err)
		return
	}

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		userID, _ := strconv.Atoi(record[0])
		movieID, _ := strconv.Atoi(record[1])
		rating, _ := strconv.ParseFloat(record[2], 64)

		ratings = append(ratings, Rating{
			UserID:  userID,
			MovieID: movieID,
			Rating:  rating,
		})
	}
}
