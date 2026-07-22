import { z } from 'zod'

const nullableText = z.string().nullable().optional()
const nullableTextArray = z.array(z.string()).nullable().optional()

export const dbaImageSchema = z
  .object({
    url: nullableText,
    path: nullableText,
  })
  .passthrough()

export const dbaLabelSchema = z
  .object({
    id: nullableText,
    text: nullableText,
    type: nullableText,
  })
  .passthrough()

export const dbaExtraSchema = z
  .object({
    id: nullableText,
    label: nullableText,
    values: nullableTextArray,
  })
  .passthrough()

export const dbaPriceSchema = z
  .object({
    amount: z.number().int().nullable().optional(),
    currencyCode: nullableText,
    priceUnit: nullableText,
  })
  .passthrough()

export const dbaDocSchema = z
  .object({
    type: nullableText,
    id: nullableText,
    mainSearchKey: nullableText,
    heading: nullableText,
    location: nullableText,
    image: dbaImageSchema.nullable().optional(),
    flags: nullableTextArray,
    timestamp: z.number().int().nullable().optional(),
    adType: z.number().int().nullable().optional(),
    labels: z.array(dbaLabelSchema).nullable().optional(),
    canonicalUrl: nullableText,
    extras: z.array(dbaExtraSchema).nullable().optional(),
    price: dbaPriceSchema.nullable().optional(),
    distance: z.number().int().nullable().optional(),
    imageUrls: nullableTextArray,
    brand: nullableText,
    adId: z.number().int().nullable().optional(),
  })
  .passthrough()

export const dbaSearchResponseSchema = z.array(dbaDocSchema)

export const dbaSearchRequestSchema = z.object({
  limit: z.number().int().positive().optional(),
  page: z.number().int().positive().optional(),
  queries: z.array(z.string().trim().min(1)).min(1),
})

export const dbaSearchFormSchema = z
  .object({
    search: z.string().trim().min(1, 'Enter at least one search term.'),
    limit: z.coerce.number().int().min(1).max(50).default(20),
    page: z.coerce.number().int().min(1).default(1),
  })
  .transform(({ search, limit, page }) => ({
    limit,
    page,
    queries: search
      .split(',')
      .map((query) => query.trim())
      .filter(Boolean),
  }))

export type DbaImage = z.infer<typeof dbaImageSchema>
export type DbaLabel = z.infer<typeof dbaLabelSchema>
export type DbaExtra = z.infer<typeof dbaExtraSchema>
export type DbaPrice = z.infer<typeof dbaPriceSchema>
export type DbaDoc = z.infer<typeof dbaDocSchema>
export type DbaSearchResponse = z.infer<typeof dbaSearchResponseSchema>
export type DbaSearchRequest = z.infer<typeof dbaSearchRequestSchema>
export type DbaSearchFormInput = z.input<typeof dbaSearchFormSchema>
export type DbaSearchFormRequest = z.output<typeof dbaSearchFormSchema>