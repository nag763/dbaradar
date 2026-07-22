import { queryOptions, useQuery } from '@tanstack/react-query'

import { dbaApiClient } from './client'
import {
  dbaSearchRequestSchema,
  dbaSearchResponseSchema,
  type DbaSearchRequest,
} from '../schemas/dba'

export async function searchDba(request: DbaSearchRequest) {
  const parsedRequest = dbaSearchRequestSchema.parse(request)
  const { data } = await dbaApiClient.post('/dba/search', parsedRequest)

  return dbaSearchResponseSchema.parse(data)
}

export function dbaSearchQueryOptions(
  request: DbaSearchRequest | null,
  requestNonce = 0,
  autoRefresh = false,
) {
  return queryOptions({
    queryKey: [
      'dba',
      'search',
      requestNonce,
      request?.queries.join('|') ?? '',
      request?.limit ?? null,
      request?.page ?? null,
    ] as const,
    queryFn: async () => {
      if (!request) {
        throw new Error('Missing DBA search request.')
      }

      return searchDba(request)
    },
    enabled: Boolean(request?.queries.length),
    staleTime: 30_000,
    refetchInterval: autoRefresh ? 120_000 : false,
  })
}

export function useDbaSearchQuery(
  request: DbaSearchRequest | null,
  requestNonce = 0,
  autoRefresh = false,
) {
  return useQuery(dbaSearchQueryOptions(request, requestNonce, autoRefresh))
}