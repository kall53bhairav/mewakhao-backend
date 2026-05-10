package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Schema struct {
	Environment        string `env:"ENV"`
	HttpPort           int    `env:"PORT"`
	DatabaseURI        string `env:"DATABASE_URL"`
	AuthSecret         string `env:"AUTH_SECRET"`
	RazorpayKeyID      string `env:"RAZORPAY_KEY_ID"`
	RazorpayKeySecret  string `env:"RAZORPAY_KEY_SECRET"`
	ShiprocketEmail    string `env:"SHIPROCKET_EMAIL"`
	ShiprocketPassword string `env:"SHIPROCKET_PASSWORD"`
	SMTPHost           string `env:"SMTP_HOST"`
	SMTPPort           int    `env:"SMTP_PORT"`
	SMTPUser           string `env:"SMTP_USER"`
	SMTPPass           string `env:"SMTP_PASS"`
	SMTPFrom           string `env:"SMTP_FROM"`
	FrontendURL        string `env:"FRONTEND_URL"`
}

const (
	ProductionEnv = "production"

	DatabaseTimeout    = 5 * time.Second
	ProductCachingTime = 1 * time.Minute
)

var (
	cfg Schema
)

func LoadConfig() *Schema {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error on load configuration file, error: %v", err)
	}
	cfg.DatabaseURI = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURI == "" {
		log.Print("warning: DATABASE_URL is empty")
	}

	portStr := os.Getenv("PORT")
	if portStr == "" {
		cfg.HttpPort = 8080
	} else {
		if p, err := strconv.Atoi(portStr); err != nil {
			log.Printf("invalid PORT %q, using default 8080: %v", portStr, err)
			cfg.HttpPort = 8080
		} else {
			cfg.HttpPort = p
		}
	}

	cfg.Environment = os.Getenv("ENV")
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}

	cfg.RazorpayKeyID = os.Getenv("RAZORPAY_KEY_ID")
	cfg.RazorpayKeySecret = os.Getenv("RAZORPAY_KEY_SECRET")
	cfg.ShiprocketEmail = os.Getenv("SHIPROCKET_EMAIL")
	cfg.ShiprocketPassword = os.Getenv("SHIPROCKET_PASSWORD")

	cfg.SMTPHost = os.Getenv("SMTP_HOST")
	cfg.SMTPUser = os.Getenv("SMTP_USER")
	cfg.SMTPPass = os.Getenv("SMTP_PASS")
	cfg.SMTPFrom = os.Getenv("SMTP_FROM")
	if p, err := strconv.Atoi(os.Getenv("SMTP_PORT")); err == nil {
		cfg.SMTPPort = p
	} else {
		cfg.SMTPPort = 587 // default STARTTLS
	}

	cfg.FrontendURL = os.Getenv("FRONTEND_URL")
	if cfg.FrontendURL == "" {
		cfg.FrontendURL = "http://localhost:3000"
	}

	return &cfg
}

func GetEnv() *Schema {
	return &cfg
}
