import axios from 'axios'

export const dbaApiBaseUrl =
  import.meta.env.VITE_DBA_API_URL ?? 'http://localhost:8080'

export const dbaApiClient = axios.create({
  baseURL: dbaApiBaseUrl,
  headers: {
    'Content-Type': 'application/json',
  },
})

export function resolveDbaUrl(path: string) {
  return new URL(path, dbaApiBaseUrl).toString()
}