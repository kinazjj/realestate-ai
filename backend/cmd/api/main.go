package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"realestate-ai/backend/internal/api/handlers"
	"realestate-ai/backend/internal/models"
	"realestate-ai/backend/internal/pkg/ai"
	"realestate-ai/backend/internal/repository"
	"realestate-ai/backend/internal/service"
)

func main() {
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=require",
			getEnv("DB_HOST", "localhost"),
			getEnv("DB_USER", "postgres"),
			getEnv("DB_PASSWORD", "postgres"),
			getEnv("DB_NAME", "realestate"),
			getEnv("DB_PORT", "5432"),
		)
	}

	var db *gorm.DB
	var err error
	for i := 0; i < 5; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("DB connection attempt %d failed: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	err = db.AutoMigrate(&models.Lead{}, &models.Property{}, &models.ChatMessage{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	seedProperties(db)

	aiRouter := ai.NewRouterFromEnv()
	leadRepo := repository.NewLeadRepository(db)
	propRepo := repository.NewPropertyRepository(db)
	chatRepo := repository.NewChatRepository(db)
	telegramSvc := service.NewTelegramService(aiRouter, leadRepo, propRepo, chatRepo)
	handler := handlers.NewHandler(leadRepo, propRepo, chatRepo, telegramSvc, aiRouter)

	r := gin.Default()
	r.Use(corsMiddleware())
	handler.RegisterRoutes(r)

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func seedProperties(db *gorm.DB) {
	var count int64
	db.Model(&models.Property{}).Count(&count)
	if count > 0 {
		return
	}

	properties := []models.Property{
		{City: "الرياض", Price: 850000, Type: "فيلا", Bedrooms: 5, Bathrooms: 4, AreaSqm: 350, Description: "فيلا فاخرة في حي الياسمين، حديقة خاصة ومجلس كبير"},
		{City: "الرياض", Price: 450000, Type: "شقة", Bedrooms: 3, Bathrooms: 2, AreaSqm: 140, Description: "شقة عصرية في برج سكني، إطلالة بانورامية"},
		{City: "جدة", Price: 1200000, Type: "فيلا", Bedrooms: 6, Bathrooms: 5, AreaSqm: 500, Description: "قصر سكني على البحر الأحمر، مسبح خاص ومدخل مستقل"},
		{City: "جدة", Price: 320000, Type: "شقة", Bedrooms: 2, Bathrooms: 2, AreaSqm: 110, Description: "شقة قريبة من الكورنيش، مناسبة للإيجار السياحي"},
		{City: "الدمام", Price: 600000, Type: "فيلا", Bedrooms: 4, Bathrooms: 3, AreaSqm: 280, Description: "فيلا في حي الفيصلية، قريبة من الخدمات والمدارس"},
		{City: "الدمام", Price: 250000, Type: "شقة", Bedrooms: 2, Bathrooms: 1, AreaSqm: 90, Description: "شقة اقتصادية للعائلات الصغيرة، قسط مريح"},
		{City: "الرياض", Price: 1800000, Type: "تجاري", Bedrooms: 0, Bathrooms: 2, AreaSqm: 400, Description: "مبنى تجاري على طريق الملك فهد، مواقف واسعة"},
		{City: "جدة", Price: 950000, Type: "فيلا", Bedrooms: 4, Bathrooms: 3, AreaSqm: 300, Description: "فيلا حديثة في حي الشاطئ، تكييف مركزي"},
	}

	for _, p := range properties {
		db.Create(&p)
	}
	log.Println("Seeded sample properties")
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
