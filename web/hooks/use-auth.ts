import { useAuth as useClerkAuth } from "@clerk/nextjs"
import { useQuery } from "@tanstack/react-query"

export function useAuthToken() {
  const { getToken } = useClerkAuth()

  return useQuery({
    queryKey: ["auth-token"],
    queryFn: async () => {
      const token = await getToken()
      if (!token) throw new Error("Not authenticated")
      return token
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
  })
}

