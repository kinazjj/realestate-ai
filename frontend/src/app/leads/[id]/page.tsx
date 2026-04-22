"use client"

import useSWR from "swr"
import { useParams } from "next/navigation"
import Link from "next/link"
import { fetcher } from "@/lib/api"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ArrowLeft, User, Phone, MapPin, Home, Calendar, MessageSquare } from "lucide-react"

interface Lead {
  id: number
  name: string
  phone: string
  city: string
  property_type: string
  timeline: string
  score: number
  tag: string
}

interface ChatMessage {
  id: number
  lead_id: number
  role: string
  content: string
  created_at: string
}

export default function LeadDetailPage() {
  const { id } = useParams()
  const { data: lead } = useSWR<Lead>(`/api/leads/${id}`, fetcher)
  const { data: messages } = useSWR<ChatMessage[]>(`/api/leads/${id}/messages`, fetcher)

  if (!lead) {
    return <div className="text-muted-foreground">Loading lead...</div>
  }

  const tagColors: Record<string, string> = {
    Serious: "bg-blue-500/10 text-blue-500",
    Urgent: "bg-red-500/10 text-red-500",
    Curious: "bg-yellow-500/10 text-yellow-500",
    Investor: "bg-purple-500/10 text-purple-500",
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link href="/leads">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4 mr-1" />
            Back
          </Button>
        </Link>
        <div>
          <h2 className="text-2xl font-bold">{lead.name || "Unknown Lead"}</h2>
          <p className="text-muted-foreground">Lead #{lead.id}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Lead Info</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center gap-3">
              <User className="h-4 w-4 text-muted-foreground" />
              <div>
                <p className="text-sm font-medium">{lead.name || "—"}</p>
                <p className="text-xs text-muted-foreground">Name</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Phone className="h-4 w-4 text-muted-foreground" />
              <div>
                <p className="text-sm font-medium">{lead.phone || "—"}</p>
                <p className="text-xs text-muted-foreground">Phone</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <MapPin className="h-4 w-4 text-muted-foreground" />
              <div>
                <p className="text-sm font-medium">{lead.city || "—"}</p>
                <p className="text-xs text-muted-foreground">City</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Home className="h-4 w-4 text-muted-foreground" />
              <div>
                <p className="text-sm font-medium">{lead.property_type || "—"}</p>
                <p className="text-xs text-muted-foreground">Property Type</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Calendar className="h-4 w-4 text-muted-foreground" />
              <div>
                <p className="text-sm font-medium">{lead.timeline || "—"}</p>
                <p className="text-xs text-muted-foreground">Timeline</p>
              </div>
            </div>
            <div className="pt-2 border-t border-border">
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">Score</span>
                <span className="font-bold">{lead.score}/100</span>
              </div>
              <div className="mt-2 h-2 w-full rounded-full bg-muted overflow-hidden">
                <div className="h-full bg-primary rounded-full" style={{ width: `${lead.score}%` }} />
              </div>
            </div>
            <Badge className={tagColors[lead.tag] || "bg-muted"}>{lead.tag}</Badge>
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle className="text-sm flex items-center gap-2">
              <MessageSquare className="h-4 w-4" />
              Chat Transcript
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4 max-h-[600px] overflow-y-auto pr-2">
              {messages && messages.length > 0 ? (
                messages.map((msg: ChatMessage) => (
                  <div
                    key={msg.id}
                    className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}
                  >
                    <div
                      className={`max-w-[80%] rounded-lg px-4 py-2 text-sm ${
                        msg.role === "user"
                          ? "bg-primary text-primary-foreground"
                          : "bg-muted"
                      }`}
                    >
                      <p>{msg.content}</p>
                      <p className="text-[10px] opacity-70 mt-1">
                        {new Date(msg.created_at).toLocaleTimeString()}
                      </p>
                    </div>
                  </div>
                ))
              ) : (
                <p className="text-sm text-muted-foreground text-center py-8">No messages yet</p>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
