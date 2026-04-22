package repository

import (
	"realestate-ai/backend/internal/models"
	"time"

	"gorm.io/gorm"
)

type LeadRepository struct {
	DB *gorm.DB
}

func NewLeadRepository(db *gorm.DB) *LeadRepository {
	return &LeadRepository{DB: db}
}

func (r *LeadRepository) Create(lead *models.Lead) error {
	return r.DB.Create(lead).Error
}

func (r *LeadRepository) Update(lead *models.Lead) error {
	return r.DB.Save(lead).Error
}

func (r *LeadRepository) GetByTelegramChatID(chatID int64) (*models.Lead, error) {
	var lead models.Lead
	err := r.DB.Where("telegram_chat_id = ?", chatID).First(&lead).Error
	return &lead, err
}

func (r *LeadRepository) GetByID(id uint) (*models.Lead, error) {
	var lead models.Lead
	err := r.DB.First(&lead, id).Error
	return &lead, err
}

func (r *LeadRepository) List() ([]models.Lead, error) {
	var leads []models.Lead
	err := r.DB.Order("created_at desc").Find(&leads).Error
	return leads, err
}

func (r *LeadRepository) GetStats() (*models.Stats, error) {
	var total int64
	r.DB.Model(&models.Lead{}).Count(&total)

	var converted int64
	r.DB.Model(&models.Lead{}).Where("status = ?", "converted").Count(&converted)

	var activeToday int64
	today := time.Now().Truncate(24 * time.Hour)
	r.DB.Model(&models.Lead{}).Where("updated_at >= ?", today).Count(&activeToday)

	var recent []models.Lead
	r.DB.Order("created_at desc").Limit(5).Find(&recent)

	var conversionRate float64
	if total > 0 {
		conversionRate = float64(converted) / float64(total) * 100
	}

	return &models.Stats{
		TotalLeads:       total,
		ConversionRate:   conversionRate,
		ActiveChatsToday: activeToday,
		RecentActivity:   recent,
	}, nil
}

type PropertyRepository struct {
	DB *gorm.DB
}

func NewPropertyRepository(db *gorm.DB) *PropertyRepository {
	return &PropertyRepository{DB: db}
}

func (r *PropertyRepository) Create(property *models.Property) error {
	return r.DB.Create(property).Error
}

func (r *PropertyRepository) Update(property *models.Property) error {
	return r.DB.Save(property).Error
}

func (r *PropertyRepository) Delete(id uint) error {
	return r.DB.Delete(&models.Property{}, id).Error
}

func (r *PropertyRepository) GetByID(id uint) (*models.Property, error) {
	var property models.Property
	err := r.DB.First(&property, id).Error
	return &property, err
}

func (r *PropertyRepository) List() ([]models.Property, error) {
	var properties []models.Property
	err := r.DB.Order("created_at desc").Find(&properties).Error
	return properties, err
}

func (r *PropertyRepository) Match(city string, maxBudget float64) ([]models.Property, error) {
	var properties []models.Property
	err := r.DB.Where("city ILIKE ? AND price <= ?", "%"+city+"%", maxBudget).
		Order("price desc").
		Limit(3).
		Find(&properties).Error
	return properties, err
}

type ChatRepository struct {
	DB *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{DB: db}
}

func (r *ChatRepository) Create(msg *models.ChatMessage) error {
	return r.DB.Create(msg).Error
}

func (r *ChatRepository) GetByLeadID(leadID uint) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage
	err := r.DB.Where("lead_id = ?", leadID).Order("created_at asc").Find(&messages).Error
	return messages, err
}
