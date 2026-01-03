package main

import (
	"database/sql"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"text/template"

	"github.com/chrisbalmer/snippetbox/internal/models"
	_ "github.com/go-sql-driver/mysql"
)

type application struct {
	logger        *slog.Logger
	debug         bool
	snippets      *models.SnippetModel
	templateCache map[string]*template.Template
}

func main() {
	addr := flag.String("addr", ":4000", "HTTP network address")
	debug := flag.Bool("debug", false, "enable debug mode")
	dsn := flag.String("dsn", "web:pass@/snippetbox?parseTime=true", "MySQL data source name")
	flag.Parse()

	handlerOptions := &slog.HandlerOptions{Level: slog.LevelInfo}
	if *debug {
		handlerOptions.Level = slog.LevelDebug
		handlerOptions.AddSource = true
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, handlerOptions))

	db, err := openDB(*dsn)
	if err != nil {
		logger.Error("unable to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connection pool established")

	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error("unable to create template cache", slog.String("error", err.Error()))
		os.Exit(1)
	}

	app := &application{
		logger:   logger,
		debug:    *debug,
		snippets: &models.SnippetModel{DB: db},
		templateCache: templateCache,
	}

	logger.Info("starting server", slog.String("addr", *addr))

	err = http.ListenAndServe(*addr, app.routes())
	logger.Error(err.Error())
	os.Exit(1)
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
