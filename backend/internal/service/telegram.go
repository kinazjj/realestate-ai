package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"realestate-ai/backend/internal/models"
	"realestate-ai/backend/internal/pkg/ai"
	"realestate-ai/backend/internal/repository"
)

type ConversationState string

const (
	StateWelcome    ConversationState = "welcome"
	StateExtracting ConversationState = "extracting"
	StateScoring    ConversationState = "scoring"
	StateMatching   ConversationState = "matching"
	StateFollowup   ConversationState = "followup"
)

type UserSession struct {
	ChatID    int64
	State     ConversationState
	Lead      *models.Lead
	Messages  []ai.Message
	UpdatedAt time.Time
}

type TelegramService struct {
	AI       *ai.Router
	LeadRepo *repository.LeadRepository
	PropRepo *repository.PropertyRepository
	ChatRepo *repository.ChatRepository
	sessions map[int64]*UserSession
	mu       sync.RWMutex
}

func NewTelegramService(ai *ai.Router, lr *repository.LeadRepository, pr *repository.PropertyRepository, cr *repository.ChatRepository) *TelegramService {
	return &TelegramService{
		AI:       ai,
		LeadRepo: lr,
		PropRepo: pr,
		ChatRepo: cr,
		sessions: make(map[int64]*UserSession),
	}
}

func (s *TelegramService) HandleMessage(ctx context.Context, chatID int64, text string, userName string) (string, error) {
	s.mu.Lock()
	session, exists := s.sessions[chatID]
	if !exists {
		lead := &models.Lead{
			TelegramChatID: chatID,
			Name:           userName,
			Status:         "new",
		}
		_ = s.LeadRepo.Create(lead)
		session = &UserSession{
			ChatID:    chatID,
			State:     StateWelcome,
			Lead:      lead,
			Messages:  []ai.Message{},
			UpdatedAt: time.Now(),
		}
		s.sessions[chatID] = session
	}
	session.UpdatedAt = time.Now()
	s.mu.Unlock()

	_ = s.ChatRepo.Create(&models.ChatMessage{
		LeadID:  session.Lead.ID,
		Role:    "user",
		Content: text,
	})

	var reply string
	var err error

	switch session.State {
	case StateWelcome:
		reply, err = s.handleWelcome(ctx, session, text)
	case StateExtracting:
		reply, err = s.handleExtracting(ctx, session, text)
	case StateScoring:
		reply, err = s.handleScoring(ctx, session)
	case StateMatching:
		reply, err = s.handleMatching(ctx, session)
	case StateFollowup:
		reply, err = s.handleFollowup(ctx, session, text)
	}

	if err == nil && reply != "" {
		_ = s.ChatRepo.Create(&models.ChatMessage{
			LeadID:  session.Lead.ID,
			Role:    "assistant",
			Content: reply,
		})
	}

	return reply, err
}

func (s *TelegramService) handleWelcome(ctx context.Context, session *UserSession, text string) (string, error) {
	prompt := `أنت مندوب مبيعات محترف في شركة عقارات فاخرة. تحدث باللهجة العربية الفصحى المهذبة.
رحب بالعميل واسأله عن:
1. المدينة التي يبحث فيها
2. ميزانيته
3. نوع العقار (شقة، فيلا، تجاري)
4. مدة الشراء المتوقعة
5. رقم الهاتف

رد بجملة قصيرة ومهذبة فقط (لا تتجاوز 3 أسطر).`

	messages := append(session.Messages, ai.Message{Role: "user", Content: text})
	resp, err := s.AI.Chat(ctx, append([]ai.Message{{Role: "system", Content: prompt}}, messages...))
	if err != nil {
		return s.fallbackWelcome(), nil
	}

	session.Messages = append(messages, ai.Message{Role: "assistant", Content: resp})
	session.State = StateExtracting
	return resp, nil
}

func (s *TelegramService) handleExtracting(ctx context.Context, session *UserSession, text string) (string, error) {
	prompt := `أنت محلل بيانات عقارية. من النص التالي، استخرج البيانات ورد بنتيجة JSON فقط بدون أي تفسير:
{"city":"","budget":0,"property_type":"","timeline":"","phone":"","complete":false}

complete=true فقط إذا امتلكت city, budget, property_type, phone.
- budget يجب أن يكون رقم بالدولار
- timeline: "فوري" أو "شهر" أو "3_أشهر" أو "6_أشهر" أو "سنة" أو "غير_محدد"
- property_type: "شقة" أو "فيلا" أو "تجاري" أو "أرض"
- phone: رقم الهاتف المستخرج

النص: ` + text

	resp, err := s.AI.Chat(ctx, []ai.Message{{Role: "user", Content: prompt}})
	if err != nil {
		return "شكراً! هل يمكنك إعطائي المزيد من التفاصيل؟", nil
	}

	extracted := s.parseExtraction(resp)
	if extracted.City != "" {
		session.Lead.City = extracted.City
	}
	if extracted.Budget > 0 {
		session.Lead.Budget = extracted.Budget
	}
	if extracted.PropertyType != "" {
		session.Lead.PropertyType = extracted.PropertyType
	}
	if extracted.Timeline != "" {
		session.Lead.Timeline = extracted.Timeline
	}
	if extracted.Phone != "" {
		session.Lead.Phone = extracted.Phone
	}

	_ = s.LeadRepo.Update(session.Lead)

	if extracted.Complete {
		session.State = StateScoring
		return s.handleScoring(ctx, session)
	}

	followUpPrompt := `أنت مندوب مبيعات. العميل لم يعطِ كل البيانات. اطلب البيانات الناقصة فقط بأدب.
البيانات الموجودة: مدينة=` + session.Lead.City + `, ميزانية=` + fmt.Sprintf("%.0f", session.Lead.Budget) + `, نوع=` + session.Lead.PropertyType + `, هاتف=` + session.Lead.Phone + `
رد بجملة قصيرة (سطرين كحد أقصى).`

	followUp, _ := s.AI.Chat(ctx, []ai.Message{{Role: "user", Content: followUpPrompt}})
	if followUp == "" {
		followUp = "هل يمكنك مشاركة باقي التفاصيل؟"
	}
	session.Messages = append(session.Messages, ai.Message{Role: "user", Content: text}, ai.Message{Role: "assistant", Content: followUp})
	return followUp, nil
}

func (s *TelegramService) handleScoring(ctx context.Context, session *UserSession) (string, error) {
	prompt := fmt.Sprintf(`أنت محلل مبيعات عقارية. حلل المحادثة التالية ورد بنتيجة JSON فقط بدون أي تفسير:
{"score":0,"tag":""}

score: من 1 إلى 100 بناءً على:
- ميزانية محددة (+30)
- مدينة محددة (+20)
- نوع عقار محدد (+15)
- هاتف محدد (+20)
- timeline "فوري" أو "شهر" (+15)

tag واحد من: "Serious", "Urgent", "Curious", "Investor"

المحادثة:
%s`, formatMessages(session.Messages))

	resp, err := s.AI.Chat(ctx, []ai.Message{{Role: "user", Content: prompt}})
	if err != nil {
		session.Lead.Score = 50
		session.Lead.Tag = "Curious"
	} else {
		score, tag := s.parseScoring(resp)
		session.Lead.Score = score
		session.Lead.Tag = tag
	}

	_ = s.LeadRepo.Update(session.Lead)
	session.State = StateMatching
	return s.handleMatching(ctx, session)
}

func (s *TelegramService) handleMatching(ctx context.Context, session *UserSession) (string, error) {
	props, err := s.PropRepo.Match(session.Lead.City, session.Lead.Budget)
	if err != nil {
		return "عذراً، حدث خطأ أثناء البحث. سأعود لمساعدتك لاحقاً.", err
	}

	if len(props) == 0 {
		reply := fmt.Sprintf("لم أجد عقارات في %s ضمن ميزانية %.0f$. هل ترغب برفع الميزانية أو تغيير المدينة؟", session.Lead.City, session.Lead.Budget)
		session.State = StateFollowup
		return reply, nil
	}

	best := props[0]
	reply := fmt.Sprintf("وجدت %d عقاراً في %s يطابق ميزانيتك %.0f$. \n\n🏆 الأفضل:\n%s\n💰 السعر: %.0f$\n🛏️ غرف: %d | 🛁 حمامات: %d\n📐 المساحة: %.0f م²\n\nهل تريد جولة افتراضية أو حجز موعد؟",
		len(props), session.Lead.City, session.Lead.Budget,
		best.Description, best.Price, best.Bedrooms, best.Bathrooms, best.AreaSqm)

	session.State = StateFollowup
	return reply, nil
}

func (s *TelegramService) handleFollowup(ctx context.Context, session *UserSession, text string) (string, error) {
	prompt := `أنت مندوب مبيعات محترف. رد على استفسار العميل بأدب وإقناع.
الرد يجب أن يكون بالعربية الفصحى، قصير (3 أسطر كحد أقصى)، ومحفز على الحجز أو الاتصال.`

	messages := append(session.Messages, ai.Message{Role: "user", Content: text})
	resp, err := s.AI.Chat(ctx, append([]ai.Message{{Role: "system", Content: prompt}}, messages...))
	if err != nil {
		return "شكراً لتواصلك. سيتصل بك أحد مستشارينا قريباً.", nil
	}

	session.Messages = append(messages, ai.Message{Role: "assistant", Content: resp})
	return resp, nil
}

func (s *TelegramService) fallbackWelcome() string {
	return "مرحباً بك! 🏡\nأنا مستشارك العقاري.\nللمساعدة، أخبرني بالمدينة والميزانية ونوع العقار المطلوب."
}

type extractedData struct {
	City         string  `json:"city"`
	Budget       float64 `json:"budget"`
	PropertyType string  `json:"property_type"`
	Timeline     string  `json:"timeline"`
	Phone        string  `json:"phone"`
	Complete     bool    `json:"complete"`
}

func (s *TelegramService) parseExtraction(raw string) extractedData {
	re := regexp.MustCompile(`(?s)\{.*\}`)
	match := re.FindString(raw)
	if match == "" {
		return extractedData{}
	}
	var data extractedData
	_ = json.Unmarshal([]byte(match), &data)

	arabicNums := map[string]string{
		"٠": "0", "١": "1", "٢": "2", "٣": "3", "٤": "4",
		"٥": "5", "٦": "6", "٧": "7", "٨": "8", "٩": "9",
	}
	for ar, en := range arabicNums {
		data.Phone = strings.ReplaceAll(data.Phone, ar, en)
	}

	return data
}

func (s *TelegramService) parseScoring(raw string) (int, string) {
	re := regexp.MustCompile(`(?s)\{.*\}`)
	match := re.FindString(raw)
	if match == "" {
		return 50, "Curious"
	}
	var result struct {
		Score int    `json:"score"`
		Tag   string `json:"tag"`
	}
	_ = json.Unmarshal([]byte(match), &result)

	if result.Score < 1 || result.Score > 100 {
		result.Score = 50
	}
	validTags := map[string]bool{"Serious": true, "Urgent": true, "Curious": true, "Investor": true}
	if !validTags[result.Tag] {
		result.Tag = "Curious"
	}
	return result.Score, result.Tag
}

func formatMessages(msgs []ai.Message) string {
	var sb strings.Builder
	for _, m := range msgs {
		sb.WriteString(fmt.Sprintf("%s: %s\n", m.Role, m.Content))
	}
	return sb.String()
}

func parseBudget(text string) float64 {
	re := regexp.MustCompile(`(\d[\d,\.\s]*)`)
	matches := re.FindAllString(text, -1)
	for _, m := range matches {
		m = strings.ReplaceAll(m, ",", "")
		m = strings.ReplaceAll(m, " ", "")
		m = strings.ReplaceAll(m, "٫", ".")
		if val, err := strconv.ParseFloat(m, 64); err == nil && val > 1000 {
			return val
		}
	}
	return 0
}
