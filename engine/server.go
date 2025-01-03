package engine

import (
	"html/template"
	"net/http"

	accounts "github.com/geofpwhite/html_games_engine/accountDB"
	connectthedots "github.com/geofpwhite/html_games_engine/connectTheDots"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"
	tictactoe "github.com/geofpwhite/html_games_engine/ticTacToe"

	connect4 "github.com/geofpwhite/html_games_engine/connect4"
	hangman "github.com/geofpwhite/html_games_engine/hangman"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func mod(a, b int) int {
	return a % b
}

// nastiest part of the system.
func Serve(inputChannel chan interfaces.Input, games map[string]interfaces.Game, playerHashes map[string]*websocket.Conn) {

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	r := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	r.SetFuncMap(template.FuncMap{"mod": mod})

	r.LoadHTMLGlob("templates/*")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "home_page.go.tmpl", gin.H{})
	})
	r.GET("/about", func(c *gin.Context) {
		// c.HTML(http.StatusOK, "about.go.tmpl", gin.H{})
		c.Redirect(http.StatusMovedPermanently, "https://github.com/geofpwhite/html_games_engine")
	})

	hangman.HangmanRoutes(r, &upgrader, games, playerHashes, inputChannel)
	connect4.Connect4Routes(r, &upgrader, games, playerHashes, inputChannel)
	connectthedots.ConnectTheDotsRoutes(r, &upgrader, games, playerHashes, inputChannel)
	tictactoe.TicTacToeRoutes(r, &upgrader, games, playerHashes, inputChannel)
	accounts.AccountRoutes(r, accounts.NewAccountsGamesHandler())

	r.Run("0.0.0.0:8080")

}
