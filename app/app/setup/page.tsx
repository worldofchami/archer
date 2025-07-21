import { createClient } from '@/utils/supabase/server'
import { cookies } from 'next/headers'

export default async function SetupPage() {
  const cookieStore = await cookies()
  const supabase = createClient(cookieStore)

  const {
    data: { user },
  } = await supabase.auth.getUser()

  return (
    <div className="flex items-center justify-center min-h-screen">
      <div className="w-full max-w-md p-8 space-y-6 bg-white rounded-lg shadow-md">
        <h1 className="text-2xl font-bold text-center text-gray-800">User ID</h1>
        {user ? (
          <>
            <p className="text-center text-gray-600">{user.id}</p>
            <p className="text-center text-sm text-gray-500">
              Please enter this ID into the CLI to fetch your details.
            </p>
          </>
        ) : (
          <p className="text-center text-red-500">User not found.</p>
        )}
      </div>
    </div>
  )
}
