# Real Estate AI Chatbot SaaS

A production-ready, Go-powered Real Estate AI Chatbot Platform with a Next.js CRM Dashboard, multi-model AI routing (Gemini + Groq), Telegram Bot integration, Supabase/PostgreSQL storage, and a property matching engine.

## Architecture

- **Backend**: Go (Gin Gonic) with Clean Architecture
- **ORM**: GORM (PostgreSQL via Supabase)
- **Frontend**: Next.js 14+ (App Router, TypeScript, Tailwind CSS, Shadcn UI, Lucide Icons)
- **AI Layer**: Multi-model routing (Gemini 3.1 Flash Lite + Groq/GPT + Llama)
- **Messaging**: Telegram Bot API via Webhook

## Project Structure

```
/root
  /backend
    /cmd/api           (main.go entry point)
    /internal
      /api/handlers    (REST API + Telegram Webhook)
      /service         (AI Logic, Telegram State Machine)
      /repository      (GORM Database operations)
      /models          (Structs for Leads, Properties, Chats)
      /pkg/ai          (API Clients for Gemini/Groq)
  /frontend            (Next.js Dashboard)
  /docs                (API docs)
```

## Prerequisites

1. **Go 1.22+** — [Download](https://go.dev/dl/)
2. **Node.js 20+** — [Download](https://nodejs.org/)
3. **Supabase PostgreSQL** — credentials already in `.env`
4. **Telegram Bot** — token already in `.env`
5. **AI API Keys** — Gemini & Groq keys already in `.env`

## Quick Start (No Docker)

### 1. Backend

```bash
cd backend

# Download dependencies
go mod tidy

# IMPORTANT: Replace the database password placeholder in .env
# Open backend/.env and replace [YOUR-PASSWORD] in DATABASE_URL with your actual Supabase password

# Run the server
go run cmd/api/main.go
```

Server starts on **http://localhost:8080**

API Endpoints:
- `GET /health` — Health check
- `GET /api/stats` — Dashboard stats
- `GET /api/leads` — List all leads
- `GET /api/leads/:id` — Get lead details
- `GET /api/leads/:id/messages` — Chat transcript
- `GET /api/properties` — List properties
- `POST /api/properties` — Create property
- `PUT /api/properties/:id` — Update property
- `DELETE /api/properties/:id` — Delete property
- `POST /api/webhook` — Telegram Bot Webhook

### 2. Frontend

```bash
cd frontend

# Install dependencies
npm install

# Run dev server
npm run dev
```

Dashboard opens on **http://localhost:3000**

### 3. Telegram Bot Setup

1. Open Telegram and find your bot using the token provided in `.env`
2. Set the webhook URL to your backend:
   ```bash
   curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
     -H "Content-Type: application/json" \
     -d '{"url": "https://your-ngrok-url/api/webhook"}'
   ```
   For local testing, use [ngrok](https://ngrok.com/):
   ```bash
   ngrok http 8080
   ```

## Features

### AI Router & Failover
- Provider interface with Gemini and Groq implementations
- Automatic key rotation across 5 Gemini keys and 2 Groq keys
- Failover: if one key/model fails, automatically retries the next

### Telegram Bot (Arabic)
- State machine: `welcome` → `extracting` → `scoring` → `matching` → `followup`
- Extracts: City, Budget, Property Type, Timeline, Phone
- AI-powered lead scoring (1-100) with tags: Serious, Urgent, Curious, Investor
- Property matching engine: returns top 3 matching listings

### CRM Dashboard
- **Overview**: Total leads, conversion rate, active chats, recent activity
- **Leads Table**: Searchable table with Score, Tag, Status, View Transcript
- **Live Chat Viewer**: Full conversation history per lead
- **Property Management**: CRUD for real estate listings

## Environment Variables

Already configured in `.env` files. Key variables:

| Variable | Description |
|----------|-------------|
| `TELEGRAM_BOT_TOKEN` | Bot token from BotFather |
| `GEMINI_KEYS` | JSON array of 5 Gemini API keys |
| `GROQ_KEYS` | JSON array of 2 Groq API keys |
| `DATABASE_URL` | Supabase PostgreSQL connection string |

## Database Schema

GORM auto-migrates on startup:
- **leads**: id, telegram_chat_id, name, phone, city, budget, property_type, timeline, score, tag, status
- **properties**: id, city, price, type, bedrooms, bathrooms, area_sqm, description, image_url
- **chat_messages**: id, lead_id, role, content, created_at

## Seed Data

8 sample properties are automatically seeded on first run (Riyadh, Jeddah, Dammam).

## Tech Stack

- Go 1.22, Gin, GORM, godotenv
- Next.js 14, React 18, TypeScript, Tailwind CSS
- Shadcn UI components, Lucide icons, Recharts, SWR
- Supabase PostgreSQL
- Gemini 3.1 Flash Lite, Groq (GPT-oss-120b, Llama-3.3-70b)

## Notes

- All bot replies are in **Arabic** (professional/formal)
- Dashboard UI is in **English**
- Dark theme configured by default
- CORS enabled for local development
