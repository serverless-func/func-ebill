package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type fetchConfig struct {
	Username string `form:"username"`
	Password string `form:"password"`
	Hour     int64  `form:"hour"`
}

func main() {
	r := gin.New()
	r.Use(gin.Recovery())

	setupRouter(r)

	start(&http.Server{
		Addr:    fmt.Sprintf(":%s", getenv("FC_SERVER_PORT", "9000")),
		Handler: r,
	})
}

func setupRouter(r *gin.Engine) {
	rg := r.Group("/ebill")
	rg.POST("/cmb", func(c *gin.Context) {
		var cfg fetchConfig
		if c.ShouldBindJSON(&cfg) != nil {
			c.JSON(http.StatusOK, failed("missing required body"))
			return
		}
		if cfg.Hour == 0 {
			cfg.Hour = 24
		}

		orders, err := emailParseCmb(cfg)
		if err != nil {
			c.JSON(http.StatusOK, failed(err.Error()))
			return
		}

		c.JSON(http.StatusOK, data(orders))
	})

	rg.POST("/file/cmb", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusOK, failed(err.Error()))
			return
		}
		filename := "/tmp/" + filepath.Base(file.Filename)
		if err := c.SaveUploadedFile(file, filename); err != nil {
			c.JSON(http.StatusOK, failed(err.Error()))
			return
		}
		defer func() {
			_ = os.Remove(filename)
		}()

		orders, err := fileParseCmb(filename)
		if err != nil {
			c.JSON(http.StatusOK, failed(err.Error()))
			return
		}

		c.JSON(http.StatusOK, data(orders))
	})
}

func start(srv *http.Server) {
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("listen: %s\n", err)
		}
	}()

	log.Printf("Start Server @ %s", srv.Addr)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Print("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server Shutdown:%s", err)
	}
	<-ctx.Done()
	log.Print("Server exiting")
}

func getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func failed(msg string) gin.H {
	return gin.H{
		"msg":       msg,
		"timestamp": time.Now().Unix(),
	}
}

func data(data interface{}) gin.H {
	return gin.H{
		"msg":       "success",
		"data":      data,
		"timestamp": time.Now().Unix(),
	}
}
