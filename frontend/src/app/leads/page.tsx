"use client"

import useSWR from "swr"
import Link from "next/link"
import { fetcher, del } from "@/lib/api"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Search, Eye, Trash2 } from "lucide-react"
import { useState } from "react"

interface Lead {
  id: number
  name: string
  city: string
  budget: number
  score: number
  tag: string
  status: string
}

export default function LeadsPage() {
  const { data: leads, mutate } = useSWR<Lead[]>("/api/leads", fetcher)
  const [search, setSearch] = useState("")

  const filtered = leads?.filter((lead: Lead) =>
    (lead.name || "").toLowerCase().includes(search.toLowerCase()) ||
    (lead.city || "").toLowerCase().includes(search.toLowerCase()) ||
    (lead.tag || "").toLowerCase().includes(search.toLowerCase())
  ) || []

  const tagColors: Record<string, string> = {
    Serious: "bg-blue-500/10 text-blue-500",
    Urgent: "bg-red-500/10 text-red-500",
    Curious: "bg-yellow-500/10 text-yellow-500",
    Investor: "bg-purple-500/10 text-purple-500",
  }

  async function handleDelete(id: number) {
    if (!confirm("Delete this lead?")) return
    await del(`/api/leads/${id}`)
    mutate()
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Leads</h2>
          <p className="text-muted-foreground">Manage and qualify your real estate leads</p>
        </div>
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search leads..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 pr-4 py-2 rounded-md border border-border bg-background text-sm w-64 focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
      </div>

      <Card>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-left text-muted-foreground">
                  <th className="px-4 py-3 font-medium">Name</th>
                  <th className="px-4 py-3 font-medium">City</th>
                  <th className="px-4 py-3 font-medium">Budget</th>
                  <th className="px-4 py-3 font-medium">Score</th>
                  <th className="px-4 py-3 font-medium">Tag</th>
                  <th className="px-4 py-3 font-medium">Status</th>
                  <th className="px-4 py-3 font-medium text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((lead: Lead) => (
                  <tr key={lead.id} className="border-b border-border last:border-0 hover:bg-accent/50">
                    <td className="px-4 py-3 font-medium">{lead.name || "—"}</td>
                    <td className="px-4 py-3">{lead.city || "—"}</td>
                    <td className="px-4 py-3">{lead.budget ? `$${lead.budget.toLocaleString()}` : "—"}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <div className="h-2 w-16 rounded-full bg-muted overflow-hidden">
                          <div
                            className="h-full bg-primary rounded-full"
                            style={{ width: `${lead.score}%` }}
                          />
                        </div>
                        <span className="text-xs">{lead.score}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <Badge className={tagColors[lead.tag] || "bg-muted"}>{lead.tag}</Badge>
                    </td>
                    <td className="px-4 py-3 capitalize">{lead.status}</td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <Link href={`/leads/${lead.id}`}>
                          <Button variant="ghost" size="sm">
                            <Eye className="h-4 w-4" />
                          </Button>
                        </Link>
                        <Button variant="ghost" size="sm" onClick={() => handleDelete(lead.id)}>
                          <Trash2 className="h-4 w-4 text-red-500" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
                {filtered.length === 0 && (
                  <tr>
                    <td colSpan={7} className="px-4 py-8 text-center text-muted-foreground">
                      No leads found
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
