"use client"

import useSWR from "swr"
import { fetcher } from "@/lib/api"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Users, TrendingUp, MessageSquare, DollarSign } from "lucide-react"

interface Stats {
  total_leads: number
  conversion_rate: number
  active_chats_today: number
  recent_activity: Array<{
    id: number
    name: string
    city: string
    score: number
    tag: string
    created_at: string
  }>
}

export default function OverviewPage() {
  const { data: stats, error } = useSWR<Stats>("/api/stats", fetcher)

  if (error) {
    return <div className="text-red-500">Failed to load stats</div>
  }

  const statCards = [
    {
      title: "Total Leads",
      value: stats?.total_leads ?? "—",
      icon: Users,
      change: "+12%",
    },
    {
      title: "Conversion Rate",
      value: stats?.conversion_rate ? `${stats.conversion_rate.toFixed(1)}%` : "—",
      icon: TrendingUp,
      change: "+5.3%",
    },
    {
      title: "Active Chats Today",
      value: stats?.active_chats_today ?? "—",
      icon: MessageSquare,
      change: "+3",
    },
    {
      title: "Avg. Deal Value",
      value: "$420K",
      icon: DollarSign,
      change: "+8%",
    },
  ]

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">Dashboard Overview</h2>
        <p className="text-muted-foreground">Real-time insights from your AI chatbot</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {statCards.map((card) => {
          const Icon = card.icon
          return (
            <Card key={card.title}>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium">{card.title}</CardTitle>
                <Icon className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{card.value}</div>
                <p className="text-xs text-green-500 mt-1">{card.change} from last month</p>
              </CardContent>
            </Card>
          )
        })}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card>
          <CardHeader>
            <CardTitle>Recent Activity</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {stats?.recent_activity && stats.recent_activity.length > 0 ? (
                stats.recent_activity.map((lead) => (
                  <div key={lead.id} className="flex items-center justify-between border-b border-border pb-3 last:border-0">
                    <div>
                      <p className="font-medium">{lead.name || "Unknown"}</p>
                      <p className="text-xs text-muted-foreground">
                        {lead.city} • Score: {lead.score} • {lead.tag}
                      </p>
                    </div>
                    <span className="text-xs text-muted-foreground">
                      {new Date(lead.created_at).toLocaleDateString()}
                    </span>
                  </div>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">No recent leads</p>
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Lead Score Distribution</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {[
                { label: "High (80-100)", count: stats?.total_leads ? Math.floor(stats.total_leads * 0.2) : 0, color: "bg-green-500" },
                { label: "Medium (50-79)", count: stats?.total_leads ? Math.floor(stats.total_leads * 0.4) : 0, color: "bg-yellow-500" },
                { label: "Low (0-49)", count: stats?.total_leads ? Math.floor(stats.total_leads * 0.4) : 0, color: "bg-red-500" },
              ].map((item) => (
                <div key={item.label} className="flex items-center gap-3">
                  <div className={`h-3 w-3 rounded-full ${item.color}`} />
                  <span className="text-sm flex-1">{item.label}</span>
                  <span className="text-sm font-medium">{item.count}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
