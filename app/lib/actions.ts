"use server"

import { createClient } from "@/utils/supabase/server"
import { redirect } from "next/navigation"
import { cookies } from "next/headers"

export async function signup(formData: FormData) {
    const email = formData.get("email")
    const password = formData.get("password")

    const cookieStore = await cookies()
    const supabase = createClient(cookieStore);

    console.log(email, password)

    const { error } = await supabase.auth.signUp({
        email: email as string,
        password: password as string,
    })

    if (error) {
        console.error(error)
        throw error
    }

    redirect("/welcome")
}

export async function login(formData: FormData) {
    const email = formData.get("email")
    const password = formData.get("password")

    const cookieStore = await cookies()
    const supabase = createClient(cookieStore);

    const { data, error } = await supabase.auth.signInWithPassword({
        email: email as string,
        password: password as string,
    })

    if (error) {
        console.error(error)
        throw error
    }

    redirect("/welcome")
}