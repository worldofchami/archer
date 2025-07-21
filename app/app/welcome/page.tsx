import { Button } from "@/components/ui/button";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import Link from "next/link";
import { redirect } from "next/navigation";
import { cookies } from "next/headers";
import { createClient } from "@/utils/supabase/server";

const SERVER_URL = process.env.SERVER_URL;

export default async function Page() {
    const cookieStore = await cookies();
    const supabase = createClient(cookieStore);
    const { data: { user } } = await supabase.auth.getUser();

    if(!user) {
        redirect("/login");
    }

    const loginUrl = `${SERVER_URL}/login?user_id=${user.id}`;

    return (
        <div className="flex items-center justify-center min-h-screen bg-background">
            <main className="flex items-center justify-center w-full">
                <Card className="w-[450px] text-center">
                    <CardHeader>
                        <CardTitle className="text-2xl">
                            Link Your Spotify Account
                        </CardTitle>
                        <CardDescription>
                            To get started with Archer, you need to connect your
                            Spotify account.
                        </CardDescription>
                    </CardHeader>
                    <CardContent className="flex flex-col items-center">
                        <p className="mb-4 text-sm text-muted-foreground">
                            Welcome, {user.email}!
                        </p>
                        <p className="mb-4 text-sm text-muted-foreground">
                            Click the button below to authorize Archer.
                        </p>
                        <Button asChild>
                            <Link href={loginUrl}>
                                Connect with Spotify
                            </Link>
                        </Button>
                        <p className="mt-4 text-xs text-muted-foreground">
                            This will redirect you to a Spotify authorization
                            page.
                        </p>
                    </CardContent>
                </Card>
            </main>
        </div>
    );
}
