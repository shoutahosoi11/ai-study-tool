package main

import (
    "net/http"
    "os"

    "github.com/joho/godotenv"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

func main() {
    _ = godotenv.Load()

    e := echo.New()
    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    e.Use(middleware.CORS())

    e.GET("/health", func(c echo.Context) error {
        return c.JSON(http.StatusOK, map[string]string{
            "status": "ok",
        })
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    e.Logger.Fatal(e.Start(":" + port))
}
