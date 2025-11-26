"use client"

import { useEffect, useState } from "react"
import { useAuth } from "@clerk/nextjs"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useToast } from "@/hooks/use-toast"
import { apiRequestWithAuth } from "@/lib/api"
import type { PolicyCategory, PolicyTemplate } from "@/lib/types"
import {
  Search,
  Sparkles,
  TrendingUp,
  Shield,
  Zap,
  Database,
  DollarSign,
  Settings,
  ChevronRight,
  Star,
  Award,
  Clock,
} from "lucide-react"
import Link from "next/link"

const DIFFICULTY_COLORS = {
  easy: "bg-green-100 text-green-800 border-green-200",
  medium: "bg-yellow-100 text-yellow-800 border-yellow-200",
  hard: "bg-red-100 text-red-800 border-red-200",
}

const CATEGORY_ICONS: Record<string, React.ReactNode> = {
  "Cost Control & Budget Management": <DollarSign className="h-5 w-5" />,
  "Resource Governance & Rightsizing": <Settings className="h-5 w-5" />,
  "Security & Compliance": <Shield className="h-5 w-5" />,
  "Operational Efficiency": <Zap className="h-5 w-5" />,
  "Data & Database Optimization": <Database className="h-5 w-5" />,
}

export default function PolicyLibraryPage() {
  const { getToken } = useAuth()
  const { toast } = useToast()

  const [categories, setCategories] = useState<PolicyCategory[]>([])
  const [templates, setTemplates] = useState<PolicyTemplate[]>([])
  const [filteredTemplates, setFilteredTemplates] = useState<PolicyTemplate[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedCategory, setSelectedCategory] = useState<string>("all")
  const [selectedDifficulty, setSelectedDifficulty] = useState<string>("all")

  useEffect(() => {
    fetchData()
  }, [])

  useEffect(() => {
    filterTemplates()
  }, [searchQuery, selectedCategory, selectedDifficulty, templates])

  const fetchData = async () => {
    try {
      setLoading(true)
      const token = await getToken()
      if (!token) return

      const [categoriesData, templatesData] = await Promise.all([
        apiRequestWithAuth("/api/policy-categories", token),
        apiRequestWithAuth("/api/policy-templates", token),
      ])

      setCategories(Array.isArray(categoriesData) ? categoriesData : [])
      setTemplates(Array.isArray(templatesData) ? templatesData : [])
    } catch (error) {
      console.error("Error fetching policy library:", error)
      toast({
        title: "Error",
        description: "Failed to load policy library",
        variant: "destructive",
      })
    } finally {
      setLoading(false)
    }
  }

  const filterTemplates = () => {
    let filtered = templates

    // Filter by search query
    if (searchQuery) {
      filtered = filtered.filter(
        (t) =>
          t.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          t.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
          t.businessImpact.toLowerCase().includes(searchQuery.toLowerCase())
      )
    }

    // Filter by category
    if (selectedCategory !== "all") {
      filtered = filtered.filter((t) => t.categoryId === selectedCategory)
    }

    // Filter by difficulty
    if (selectedDifficulty !== "all") {
      filtered = filtered.filter((t) => t.difficulty === selectedDifficulty)
    }

    setFilteredTemplates(filtered)
  }

  const parseJSON = (jsonString: string, fallback: any = []) => {
    try {
      return JSON.parse(jsonString)
    } catch {
      return fallback
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading policy library...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="p-8">
      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h1 className="text-3xl font-bold">Policy Template Library</h1>
            <p className="text-gray-600 mt-2">
              Browse and deploy production-ready cloud governance policies
            </p>
          </div>
          <Link href="/dashboard/recommendations">
            <Button className="gap-2">
              <Sparkles className="h-4 w-4" />
              View AI Recommendations
            </Button>
          </Link>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mt-6">
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">Total Templates</p>
                  <p className="text-2xl font-bold">{templates.length}</p>
                </div>
                <Star className="h-8 w-8 text-yellow-500" />
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">Categories</p>
                  <p className="text-2xl font-bold">{categories.length}</p>
                </div>
                <Settings className="h-8 w-8 text-blue-500" />
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">Potential Savings</p>
                  <p className="text-2xl font-bold">50-70%</p>
                </div>
                <TrendingUp className="h-8 w-8 text-green-500" />
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">Avg. Deploy Time</p>
                  <p className="text-2xl font-bold">2 mins</p>
                </div>
                <Clock className="h-8 w-8 text-purple-500" />
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Search and Filters */}
      <div className="mb-6 space-y-4">
        <div className="relative">
          <Search className="absolute left-3 top-3 h-4 w-4 text-gray-400" />
          <Input
            placeholder="Search policies by name, description, or impact..."
            className="pl-10"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>

        <div className="flex gap-4">
          <select
            className="px-4 py-2 border rounded-md"
            value={selectedDifficulty}
            onChange={(e) => setSelectedDifficulty(e.target.value)}
          >
            <option value="all">All Difficulties</option>
            <option value="easy">Easy</option>
            <option value="medium">Medium</option>
            <option value="hard">Hard</option>
          </select>
        </div>
      </div>

      {/* Category Tabs */}
      <Tabs value={selectedCategory} onValueChange={setSelectedCategory} className="space-y-6">
        <TabsList className="grid w-full grid-cols-6">
          <TabsTrigger value="all">All</TabsTrigger>
          {categories.map((category) => (
            <TabsTrigger key={category.id} value={category.id}>
              {category.icon} {category.name.split("&")[0].trim()}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value={selectedCategory} className="space-y-4">
          {filteredTemplates.length === 0 ? (
            <Card>
              <CardContent className="py-12 text-center">
                <p className="text-gray-500">No templates found matching your criteria</p>
              </CardContent>
            </Card>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {filteredTemplates.map((template) => {
                const cloudProviders = parseJSON(template.cloudProviders, [])
                const tags = parseJSON(template.tags, [])

                return (
                  <Card key={template.id} className="hover:shadow-lg transition-shadow">
                    <CardHeader>
                      <div className="flex items-start justify-between mb-2">
                        <CardTitle className="text-lg">{template.name}</CardTitle>
                        <Badge
                          variant="outline"
                          className={DIFFICULTY_COLORS[template.difficulty]}
                        >
                          {template.difficulty}
                        </Badge>
                      </div>
                      <CardDescription className="line-clamp-2">
                        {template.description}
                      </CardDescription>
                    </CardHeader>

                    <CardContent>
                      <div className="space-y-4">
                        {/* Savings */}
                        <div className="flex items-center gap-2">
                          <TrendingUp className="h-4 w-4 text-green-600" />
                          <span className="text-sm font-medium text-green-600">
                            {template.estimatedSavings}
                          </span>
                        </div>

                        {/* Cloud Providers */}
                        <div className="flex flex-wrap gap-1">
                          {cloudProviders.map((provider: string) => (
                            <Badge key={provider} variant="secondary" className="text-xs">
                              {provider.toUpperCase()}
                            </Badge>
                          ))}
                        </div>

                        {/* Tags */}
                        <div className="flex flex-wrap gap-1">
                          {tags.slice(0, 3).map((tag: string) => (
                            <Badge
                              key={tag}
                              variant="outline"
                              className="text-xs bg-blue-50 text-blue-700 border-blue-200"
                            >
                              {tag}
                            </Badge>
                          ))}
                        </div>

                        {/* Business Impact */}
                        <p className="text-sm text-gray-600 line-clamp-2">
                          {template.businessImpact}
                        </p>

                        {/* Deploy Button */}
                        <Link href={`/dashboard/policy-library/${template.id}`}>
                          <Button className="w-full gap-2">
                            View Details
                            <ChevronRight className="h-4 w-4" />
                          </Button>
                        </Link>
                      </div>
                    </CardContent>
                  </Card>
                )
              })}
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}
