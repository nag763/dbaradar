import { createFileRoute } from '@tanstack/react-router'
import { ArrowUpRight, Clock3, LoaderCircle, Search, X } from 'lucide-react'
import type { ChangeEvent, FormEvent, KeyboardEvent } from 'react'
import { useMemo, useState } from 'react'

import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import { Card, CardContent, CardHeader } from '#/components/ui/card'
import { Checkbox } from '#/components/ui/checkbox'
import { Input } from '#/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '#/components/ui/select'
import { Separator } from '#/components/ui/separator'
import { Skeleton } from '#/components/ui/skeleton'
import { resolveDbaUrl } from '#/interactions/api/client'
import { useDbaSearchQuery } from '#/interactions/api/dba'
import type { DbaDoc, DbaSearchRequest } from '#/interactions/schemas/dba'

export const Route = createFileRoute('/')({ component: Home })

function Home() {
  const [criteria, setCriteria] = useState<string[]>(['canon', 'nikon'])
  const [draftSearch, setDraftSearch] = useState('')
  const [limit, setLimit] = useState('10')
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [searchNonce, setSearchNonce] = useState(0)

  const request = useMemo<DbaSearchRequest | null>(() => {
    if (criteria.length === 0) {
      return null
    }

    return {
      queries: criteria,
      limit: Number(limit),
      page: 1,
    }
  }, [criteria, limit])

  const searchQuery = useDbaSearchQuery(request, searchNonce, autoRefresh)
  const docs = searchQuery.data ?? []
  const lastSearchedAt = searchQuery.dataUpdatedAt ? new Date(searchQuery.dataUpdatedAt) : null

  function commitDraftTerm() {
    const value = draftSearch.trim()
    if (!value) {
      return
    }

    const nextTerms = value
      .split(',')
      .map((term) => term.trim())
      .filter(Boolean)

    if (nextTerms.length === 0) {
      setDraftSearch('')
      return
    }

    setCriteria((currentCriteria) => {
      const existingTerms = new Set(currentCriteria.map(normalizeTerm))
      const merged = [...currentCriteria]

      for (const term of nextTerms) {
        const normalized = normalizeTerm(term)
        if (!existingTerms.has(normalized)) {
          merged.push(term)
          existingTerms.add(normalized)
        }
      }

      return merged
    })

    setDraftSearch('')
  }

  function handleDraftKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'Enter' || event.key === ',') {
      event.preventDefault()
      commitDraftTerm()
    }
  }

  function handleDraftChange(event: ChangeEvent<HTMLInputElement>) {
    setDraftSearch(event.target.value)
  }

  function handleSearchSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    commitDraftTerm()
  }

  function removeCriterion(termToRemove: string) {
    setCriteria((currentCriteria) =>
      currentCriteria.filter((term) => normalizeTerm(term) !== normalizeTerm(termToRemove)),
    )
  }

  return (
    <main className="page-wrap flex flex-col gap-8 py-8 md:py-12">
      <section className="island-shell overflow-hidden rounded-[28px] px-6 py-8 md:px-10 md:py-12">
        <div className="grid gap-8 lg:grid-cols-[1.15fr_0.85fr] lg:items-end">
          <div className="space-y-5">
            <Badge variant="secondary" className="w-fit px-3 py-1 text-xs uppercase tracking-[0.24em]">
              DBA search
            </Badge>
            <div className="space-y-3">
              <h1 className="display-title max-w-2xl text-balance text-4xl leading-tight md:text-6xl">
                Search with criteria pills, clear pricing, and polished cards.
              </h1>
              <p className="max-w-2xl text-base leading-7 text-sea-ink-soft md:text-lg">
                Type a term, press Enter, and build up the search one criterion at a time.
              </p>
            </div>
          </div>

          <Card className="border-border/60 bg-background/75 shadow-none backdrop-blur-sm">
            <CardHeader className="space-y-3">
              <div className="space-y-1">
                <h2 className="text-lg font-semibold">Search criteria</h2>
                <p className="text-sm text-sea-ink-soft">Press Enter or comma to add a pill.</p>
              </div>

              <form className="space-y-3" onSubmit={handleSearchSubmit}>
                <Input
                  value={draftSearch}
                  onChange={handleDraftChange}
                  onKeyDown={handleDraftKeyDown}
                  placeholder="Type a term and press Enter"
                  aria-label="Search criteria input"
                />

                <div className="flex flex-wrap gap-2">
                  {criteria.map((term) => (
                    <Badge key={term} variant="secondary" className="gap-2 rounded-full px-3 py-1.5">
                      <span>{term}</span>
                      <button
                        type="button"
                        onClick={() => removeCriterion(term)}
                        className="inline-flex size-4 items-center justify-center rounded-full transition hover:bg-background/80"
                        aria-label={`Remove ${term}`}
                      >
                        <X className="size-3" />
                      </button>
                    </Badge>
                  ))}
                </div>

                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="space-y-2">
                    <span className="text-sm font-medium text-foreground">Limit</span>
                    <Select value={limit} onValueChange={setLimit}>
                      <SelectTrigger className="w-full">
                        <SelectValue placeholder="Select limit" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="10">10</SelectItem>
                        <SelectItem value="20">20</SelectItem>
                        <SelectItem value="50">50</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-2">
                    <span className="text-sm font-medium text-foreground">Refresh automatically</span>
                    <label className="flex min-h-10 items-center gap-3 rounded-md border border-border px-3 py-2 text-sm">
                      <Checkbox
                        checked={autoRefresh}
                        onCheckedChange={(checked) => setAutoRefresh(checked === true)}
                      />
                      Every 2 minutes
                    </label>
                  </div>
                </div>

                <div className="flex items-center justify-between gap-3">
                  <Button type="submit" variant="outline" className="gap-2">
                    <Search className="size-4" />
                    Add term
                  </Button>
                  <Button
                    type="button"
                    onClick={() => setSearchNonce((value) => value + 1)}
                    disabled={criteria.length === 0}
                    className="gap-2"
                  >
                    {searchQuery.isFetching ? (
                      <LoaderCircle className="size-4 animate-spin" />
                    ) : (
                      <ArrowUpRight className="size-4" />
                    )}
                    Refresh now
                  </Button>
                </div>
              </form>
            </CardHeader>
          </Card>
        </div>
      </section>

      <section className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/60 bg-background/70 px-4 py-3 shadow-sm backdrop-blur-sm">
        <div className="space-y-1">
          <p className="text-sm font-medium text-foreground">Last searched</p>
          <p className="text-sm text-sea-ink-soft">
            {lastSearchedAt ? `${formatDateTime(lastSearchedAt)} · ${formatRelativeTime(lastSearchedAt)}` : 'No search yet'}
          </p>
        </div>

        <div className="flex flex-wrap gap-2">
          {criteria.length > 0 ? (
            criteria.map((term) => (
              <Badge key={term} variant="outline" className="rounded-full px-3 py-1">
                {term}
              </Badge>
            ))
          ) : (
            <Badge variant="outline">No criteria yet</Badge>
          )}
        </div>
      </section>

      <section className="space-y-4">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 className="display-title text-2xl md:text-3xl">Results</h2>
            <p className="text-sm text-sea-ink-soft">
              {searchQuery.isFetching
                ? 'Refreshing the latest matches.'
                : docs.length > 0
                  ? `Showing ${docs.length} listings from DBA.`
                  : 'Add a criterion to fetch results.'}
            </p>
          </div>
        </div>

        <Separator />

        {searchQuery.isLoading ? (
          <div className="grid gap-4">
            <ResultSkeleton />
            <ResultSkeleton />
            <ResultSkeleton />
          </div>
        ) : searchQuery.isError ? (
          <Card className="border-destructive/30 bg-destructive/5">
            <CardHeader>
              <h3 className="text-lg font-semibold text-destructive">Search failed</h3>
              <p className="text-sm text-sea-ink-soft">
                {searchQuery.error instanceof Error
                  ? searchQuery.error.message
                  : 'Unexpected error while querying the backend.'}
              </p>
            </CardHeader>
          </Card>
        ) : docs.length > 0 ? (
          <div className="grid gap-4">
            {docs.map((doc, index) => (
              <DbaResultCard key={doc.id ?? doc.canonicalUrl ?? `${index}`} doc={doc} />
            ))}
          </div>
        ) : (
          <Card className="border-dashed bg-background/70">
            <CardHeader>
              <h3 className="text-lg font-semibold">No results yet</h3>
              <p className="text-sm text-sea-ink-soft">Add a search criterion above to populate the list.</p>
            </CardHeader>
          </Card>
        )}
      </section>
    </main>
  )
}

function DbaResultCard({ doc }: { doc: DbaDoc }) {
  const title = doc.heading ?? doc.mainSearchKey ?? 'Untitled listing'
  const imageSource = doc.image?.url ?? doc.imageUrls?.[0] ?? null
  const publishedAt = formatDateTime(doc.timestamp)
  const relativePublished = formatRelativeTime(doc.timestamp)
  const price = formatPrice(doc.price)
  const canonicalUrl = doc.canonicalUrl ? resolveDbaUrl(doc.canonicalUrl) : null

  return (
    <Card className="overflow-hidden border-border/60 bg-background/80 shadow-[0_14px_30px_rgba(20,40,40,0.08)] backdrop-blur-sm">
      <div className="grid gap-0 lg:grid-cols-[220px_1fr]">
        <div className="p-4 lg:p-5">
          <div className="aspect-square overflow-hidden rounded-2xl bg-muted">
            {imageSource ? (
              <img src={resolveDbaUrl(imageSource)} alt={title} className="h-full w-full object-cover" />
            ) : (
              <div className="flex h-full items-center justify-center text-sm text-sea-ink-soft">
                No image provided
              </div>
            )}
          </div>
        </div>

        <CardContent className="flex flex-col gap-4 p-5 lg:p-6">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="space-y-2">
              <div className="flex flex-wrap gap-2">
                {doc.type ? <Badge variant="secondary">{doc.type}</Badge> : null}
                {doc.brand ? <Badge variant="outline">{doc.brand}</Badge> : null}
                {doc.flags?.some((flag) => (flag == 'shipping_exists')) && <Badge variant="secondary">
                    Shipping exists
                  </Badge>}
                {doc.flags?.some((flag) => (flag == 'buy_now')) && <Badge variant="secondary">
                    Buy easily
                  </Badge>}
              </div>

              <h3 className="display-title text-balance text-2xl leading-tight md:text-[1.75rem]">
                {title}
              </h3>

              <div className="flex flex-wrap items-center gap-3 text-sm text-sea-ink-soft">
                {publishedAt ? (
                  <span className="inline-flex items-center gap-1.5">
                    <Clock3 className="size-4" />
                    {publishedAt}
                  </span>
                ) : null}
                {relativePublished ? <span>{relativePublished}</span> : null}
              </div>
            </div>

            <div className="rounded-2xl border border-border/60 bg-background px-4 py-3 text-right shadow-sm">
              <p className="text-xs uppercase tracking-[0.2em] text-sea-ink-soft">Price</p>
              <p className="display-title text-2xl leading-none">{price}</p>
            </div>
          </div>

          <Separator />

          <div className="flex flex-wrap gap-2">
            {doc.labels?.map((label) => (
              <Badge key={label.id ?? label.text ?? label.type ?? title} variant="outline">
                {label.text ?? label.type ?? 'Label'}
              </Badge>
            ))}
            {doc.extras?.map((extra) => (
              <Badge key={extra.id ?? extra.label ?? title} variant="secondary">
                {extra.label ?? 'Extra'}
              </Badge>
            ))}
          </div>

          <div className="flex items-center justify-between gap-3">
            {canonicalUrl ? (
              <Button asChild variant="outline" size="sm" className="gap-2">
                <a href={canonicalUrl} target="_blank" rel="noreferrer">
                  Open listing
                  <ArrowUpRight className="size-4" />
                </a>
              </Button>
            ) : (
              <span className="text-sm text-sea-ink-soft">No public link provided</span>
            )}

            <Button className="text-xs text-sea-ink-soft"><a target="_blank" href={`https://www.dba.dk/recommerce/forsale/item/${doc?.id}`}>{doc.id ? "Check on DBA" : 'No DBA ID returned'}</a></Button>
          </div>
        </CardContent>
      </div>
    </Card>
  )
}

function ResultSkeleton() {
  return (
    <Card className="overflow-hidden border-border/60 bg-background/70 shadow-none">
      <div className="grid gap-0 lg:grid-cols-[220px_1fr]">
        <div className="p-4 lg:p-5">
          <Skeleton className="aspect-square rounded-2xl" />
        </div>
        <CardContent className="space-y-4 p-5 lg:p-6">
          <div className="flex gap-2">
            <Skeleton className="h-5 w-14 rounded-full" />
            <Skeleton className="h-5 w-20 rounded-full" />
          </div>
          <Skeleton className="h-8 w-2/3" />
          <div className="flex gap-3">
            <Skeleton className="h-5 w-28" />
            <Skeleton className="h-5 w-20" />
          </div>
          <Separator />
          <div className="flex flex-wrap gap-2">
            <Skeleton className="h-6 w-24 rounded-full" />
            <Skeleton className="h-6 w-20 rounded-full" />
            <Skeleton className="h-6 w-16 rounded-full" />
          </div>
          <Skeleton className="h-9 w-32" />
        </CardContent>
      </div>
    </Card>
  )
}

function formatDateTime(timestamp: DbaDoc['timestamp'] | Date | null) {
  if (timestamp == null) {
    return null
  }

  const value = timestamp instanceof Date ? timestamp : new Date(timestamp)

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
    hour12: false,
  }).format(value)
}

function formatRelativeTime(timestamp: DbaDoc['timestamp'] | Date | null) {
  if (timestamp == null) {
    return null
  }

  const value = timestamp instanceof Date ? timestamp.getTime() : timestamp

  const diffInSeconds = Math.round((Date.now() - value) / 1000)
  const absoluteSeconds = Math.abs(diffInSeconds)

  const units: Array<{ limit: number; divisor: number; label: Intl.RelativeTimeFormatUnit }> = [
    { limit: 60, divisor: 1, label: 'second' },
    { limit: 3600, divisor: 60, label: 'minute' },
    { limit: 86400, divisor: 3600, label: 'hour' },
    { limit: 604800, divisor: 86400, label: 'day' },
    { limit: 2592000, divisor: 604800, label: 'week' },
    { limit: 31536000, divisor: 2592000, label: 'month' },
  ]

  const selectedUnit = units.find((unit) => absoluteSeconds < unit.limit) ?? {
    divisor: 31536000,
    label: 'year' as const,
  }

  const relativeValue = Math.max(1, Math.round(absoluteSeconds / selectedUnit.divisor))
  return new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' }).format(
    diffInSeconds <= 0 ? relativeValue : -relativeValue,
    selectedUnit.label,
  )
}

function formatPrice(price: DbaDoc['price']) {
  const amount = price?.amount
  if (amount == null) {
    return 'DKK'
  }

  const currencyCode = price?.currencyCode ?? 'DKK'
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: currencyCode,
    maximumFractionDigits: 0,
  }).format(amount)
}

function normalizeTerm(term: string) {
  return term.trim().toLowerCase()
}