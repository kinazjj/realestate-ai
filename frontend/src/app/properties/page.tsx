"use client"

import useSWR from "swr"
import { useState } from "react"
import { fetcher, post, put, del } from "@/lib/api"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog"
import { Plus, Pencil, Trash2, Building2 } from "lucide-react"

interface Property {
  id: number
  city: string
  price: number
  type: string
  bedrooms: number
  bathrooms: number
  area_sqm: number
  description: string
  image_url: string
}

export default function PropertiesPage() {
  const { data: properties, mutate } = useSWR<Property[]>("/api/properties", fetcher)
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Property | null>(null)
  const [form, setForm] = useState({
    city: "",
    price: "",
    type: "شقة",
    bedrooms: "",
    bathrooms: "",
    area_sqm: "",
    description: "",
    image_url: "",
  })

  function resetForm() {
    setForm({
      city: "",
      price: "",
      type: "شقة",
      bedrooms: "",
      bathrooms: "",
      area_sqm: "",
      description: "",
      image_url: "",
    })
    setEditing(null)
  }

  function handleEdit(p: Property) {
    setEditing(p)
    setForm({
      city: p.city,
      price: String(p.price),
      type: p.type,
      bedrooms: String(p.bedrooms),
      bathrooms: String(p.bathrooms),
      area_sqm: String(p.area_sqm),
      description: p.description,
      image_url: p.image_url,
    })
    setOpen(true)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const body = {
      city: form.city,
      price: parseFloat(form.price),
      type: form.type,
      bedrooms: parseInt(form.bedrooms) || 0,
      bathrooms: parseInt(form.bathrooms) || 0,
      area_sqm: parseFloat(form.area_sqm) || 0,
      description: form.description,
      image_url: form.image_url,
    }

    if (editing) {
      await put(`/api/properties/${editing.id}`, body)
    } else {
      await post("/api/properties", body)
    }
    mutate()
    setOpen(false)
    resetForm()
  }

  async function handleDelete(id: number) {
    if (!confirm("Delete this property?")) return
    await del(`/api/properties/${id}`)
    mutate()
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Properties</h2>
          <p className="text-muted-foreground">Manage real estate listings</p>
        </div>
        <Button onClick={() => { resetForm(); setOpen(true) }}>
          <Plus className="h-4 w-4 mr-1" />
          Add Property
        </Button>
      </div>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? "Edit Property" : "Add Property"}</DialogTitle>
            <DialogDescription>Fill in the property details below.</DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4 mt-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-sm font-medium">City</label>
                <input
                  value={form.city}
                  onChange={(e) => setForm({ ...form, city: e.target.value })}
                  className="w-full mt-1 rounded-md border border-border bg-background px-3 py-2 text-sm"
                  required
                />
              </div>
              <div>
                <label className="text-sm font-medium">Price ($)</label>
                <input
                  type="number"
                  value={form.price}
                  onChange={(e) => setForm({ ...form, price: e.target.value })}
                  className="w-full mt-1 rounded-md border border-border bg-background px-3 py-2 text-sm"
                  required
                />
              </div>
            </div>
            <div>
              <label className="text-sm font-medium">Type</label>
              <select
                value={form.type}
                onChange={(e) => setForm({ ...form, type: e.target.value })}
                className="w-full mt-1 rounded-md border border-border bg-background px-3 py-2 text-sm"
              >
                <option>شقة</option>
                <option>فيلا</option>
                <option>تجاري</option>
                <option>أرض</option>
              </select>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <div>
                <label className="text-sm font-medium">Bedrooms</label>
                <input
                  type="number"
                  value={form.bedrooms}
                  onChange={(e) => setForm({ ...form, bedrooms: e.target.value })}
                  className="w-full mt-1 rounded-md border border-border bg-background px-3 py-2 text-sm"
                />
              </div>
              <div>
                <label className="text-sm font-medium">Bathrooms</label>
                <input
                  type="number"
                  value={form.bathrooms}
                  onChange={(e) => setForm({ ...form, bathrooms: e.target.value })}
                  className="w-full mt-1 rounded-md border border-border bg-background px-3 py-2 text-sm"
                />
              </div>
              <div>
                <label className="text-sm font-medium">Area (m²)</label>
                <input
                  type="number"
                  value={form.area_sqm}
                  onChange={(e) => setForm({ ...form, area_sqm: e.target.value })}
                  className="w-full mt-1 rounded-md border border-border bg-background px-3 py-2 text-sm"
                />
              </div>
            </div>
            <div>
              <label className="text-sm font-medium">Description</label>
              <textarea
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
                className="w-full mt-1 rounded-md border border-border bg-background px-3 py-2 text-sm"
                rows={3}
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
                Cancel
              </Button>
              <Button type="submit">{editing ? "Update" : "Create"}</Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {properties?.map((p: Property) => (
          <Card key={p.id}>
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">{p.city}</CardTitle>
                <div className="flex gap-1">
                  <Button variant="ghost" size="sm" onClick={() => handleEdit(p)}>
                    <Pencil className="h-3.5 w-3.5" />
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => handleDelete(p.id)}>
                    <Trash2 className="h-3.5 w-3.5 text-red-500" />
                  </Button>
                </div>
              </div>
              <p className="text-xs text-muted-foreground">{p.type}</p>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground line-clamp-2">{p.description}</p>
              <div className="mt-3 flex items-center justify-between">
                <span className="font-bold text-lg">${p.price.toLocaleString()}</span>
                <div className="flex items-center gap-3 text-xs text-muted-foreground">
                  <span>{p.bedrooms} bed</span>
                  <span>{p.bathrooms} bath</span>
                  <span>{p.area_sqm}m²</span>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
        {(!properties || properties.length === 0) && (
          <div className="col-span-full text-center py-12 text-muted-foreground">
            <Building2 className="h-12 w-12 mx-auto mb-3 opacity-50" />
            <p>No properties yet. Add your first listing.</p>
          </div>
        )}
      </div>
    </div>
  )
}
