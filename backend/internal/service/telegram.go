package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	BotToken string
	sessions map[int64]*UserSession
	mu       sync.RWMutex
}

func NewTelegramService(ai *ai.Router, lr *repository.LeadRepository, pr *repository.PropertyRepository, cr *repository.ChatRepository, botToken string) *TelegramService {
	return &TelegramService{
		AI:       ai,
		LeadRepo: lr,
		PropRepo: pr,
		ChatRepo: cr,
		BotToken: botToken,
		sessions: make(map[int64]*UserSession),
	}
}

func (s *TelegramService) SendTelegramMessage(chatID int64, text string) error {
	if s.BotToken == "" {
		return fmt.Errorf("no bot token configured")
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.BotToken)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	body, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
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

	// Process AI asynchronously to avoid Telegram webhook timeout (10s)
	go s.processMessageAsync(session, chatID, text, userName)

	// Return immediate response
	return "⏳ جاري تحليل طلبك... سأرد عليك خلال ثواني ✨", nil
}

func (s *TelegramService) processMessageAsync(session *UserSession, chatID int64, text string, userName string) {
	ctx := context.Background()

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
		_ = s.SendTelegramMessage(chatID, reply)
	} else if err != nil {
		_ = s.SendTelegramMessage(chatID, "عذراً، حدث خطأ. سأعود لمساعدتك قريباً.")
	}
}

func (s *TelegramService) handleWelcome(ctx context.Context, session *UserSession, text string) (string, error) {
	prompt := `أنت مستشار عقاري محترف في شركة كينز العقارية. اسمك "عمر". تحدث بالعربية الفصحى المهذبة والودودة كأنك صديق للعميل، ليس روبوت.

قواعد مهمة:
- إذا سأل سؤال مباشر (هل عندكم...؟ كم سعر...؟) أجب عليه مباشرة وباختصار أولاً
- ثم اسأل سؤال واحد فقط (لا تطلب كل المعلومات دفعة واحدة)
- لا تطلب رقم الهاتف في الرسالة الأولى أبداً
- كن طبيعياً وودوداً، استخدم تعبيرات مثل "بالتأكيد" "عظيم" " delighted"
- اذكر اسم الشركة "كينز العقارية" بشكل طبيعي

مثال: إذا قال "عندكم بيوت في الرياض؟" رد: "نعم بالتأكيد! لدينا تشكيلة ممتازة في الرياض. هل تفضل فيلا أم شقة؟"

رد بجملتين أو ثلاث كحد أقصى.`

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
		return "عذراً، حدث خطأ فني. لكن لا تقلق، أنا هنا! هل يمكنك إعطائي ميزانيتك التقريبية ونوع العقار اللي تبحث عنه؟", nil
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

	followUpPrompt := `أنت عمر، مستشار عقاري في كينز العقارية. تحدث بالعربية الفصحى المهذبة والودودة.

المحادثة حتى الآن:
- المدينة: ` + session.Lead.City + `
- الميزانية: ` + fmt.Sprintf("%.0f", session.Lead.Budget) + `
- نوع العقار: ` + session.Lead.PropertyType + `
- التوقيت: ` + session.Lead.Timeline + `

رد بأسلوب طبيعي وودود. لا تكرر ما يعرفه العميل بالفعل. اسأل عن المعلومات الناقصة فقط بأسلوب محادثة (مثل "عظيم! في أي نطاق ميزانية تبحث؟" أو "ممتاز! متى تخطط للشراء تقريباً؟").
لا تذكر أنك تحلل بيانات. رد بجملة واحدة أو جملتين.`

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
		session.Lead.Score = 60
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
		reply := fmt.Sprintf("🤔 حالياً ما عندنا عقار متاح في %s ضمن ميزانية %.0f$\n\nلكن عندنا خيارات رائعة قريبة:\n• نفس المدينة بميزانية أعلى\n• مدن مجاورة بأسعار منافسة\n• عقارات بالتقسيط المريح\n\nهل تبي نبحث لك بميزانية مختلفة أو مدينة ثانية؟ 🏘️", session.Lead.City, session.Lead.Budget)
		session.State = StateFollowup
		return reply, nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("✨ وجدنا %d عقار رائع في %s يطابق ميزانيتك %.0f$\n\n", len(props), session.Lead.City, session.Lead.Budget))

	for i, p := range props {
		if i >= 3 {
			break
		}
		b.WriteString(fmt.Sprintf("🏠 العقار %d\n", i+1))
		b.WriteString(fmt.Sprintf("📍 %s\n", p.Description))
		b.WriteString(fmt.Sprintf("💰 السعر: %.0f$ | 🛏️ %d غرف | 🛁 %d حمام | 📐 %.0f م²\n\n",
			p.Price, p.Bedrooms, p.Bathrooms, p.AreaSqm))
	}

	if len(props) > 3 {
		b.WriteString(fmt.Sprintf("... و %d عقار إضافي 💫\n\n", len(props)-3))
	}

	b.WriteString("📞 تبي نتواصل معك ونعطيك تفاصيل أكثر؟\nأو تحب نحجز لك موعد معاينة؟ 🏡")
	reply := b.String()

	session.State = StateFollowup
	return reply, nil
}

func (s *TelegramService) handleFollowup(ctx context.Context, session *UserSession, text string) (string, error) {
	prompt := `أنت عمر، مستشار عقاري في كينز العقارية. تحدث بالعربية الفصحى المهذبة والودودة كأنك صديق.

رد على استفسار العميل بأدب وإقناع. كن طبيعياً ولا تكرر نفس العبارات.
إذا طلب تفاصيل أكثر عن عقار، أعطِ معلومات إضافية مفيدة.
إذا طلب حجز/زيارة، رحب وأكد على أننا سنتواصل معه.
لا تطلب رقم هاتف إلا إذا كان العميل مهتم فعلاً.
رد بـ 3 أسطر كحد أقصى.`

	messages := append(session.Messages, ai.Message{Role: "user", Content: text})
	resp, err := s.AI.Chat(ctx, append([]ai.Message{{Role: "system", Content: prompt}}, messages...))
	if err != nil {
		return "شكراً لتفاعلك معنا! 💫 سيتواصل معك أحد مستشارينا خلال 24 ساعة. هل هناك شيء آخر أقدر أساعدك فيه الآن؟", nil
	}

	session.Messages = append(messages, ai.Message{Role: "assistant", Content: resp})
	return resp, nil
}

func (s *TelegramService) fallbackWelcome() string {
	return "أهلاً وسهلاً بك في كينز العقارية! 🏡\n\nأنا عمر، مستشارك العقاري. أخبرني في أي مدينة تبحث وما نوع العقار اللي يهمك، وأنا أساعدك بكل سرور."
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
