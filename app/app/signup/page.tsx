import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { signup } from "@/lib/actions"

export default async function Page() {
  return (
    <div className="flex items-center justify-center min-h-screen bg-background">
      <main className="flex items-center justify-center w-full">
        <Card className="w-[400px]">
          <CardHeader className="text-center">
            <CardTitle className="text-2xl">Archer</CardTitle>
            <CardDescription>Your command-line Spotify player</CardDescription>
          </CardHeader>
          <CardContent>
            <form action={signup} id="signup-form">
              <div className="grid w-full items-center gap-4">
                <div className="flex flex-col space-y-1.5">
                  <Label htmlFor="email">Email</Label>
                  <Input name="email" id="email" placeholder="Email" />
                </div>
                <div className="flex flex-col space-y-1.5">
                  <Label htmlFor="password">Password</Label>
                  <Input name="password" id="password" type="password" placeholder="Password" />
                </div>
              </div>
            </form>
          </CardContent>
          <CardFooter className="flex flex-col">
            <Button className="w-full" form="signup-form" type="submit">Sign Up</Button>
            <p className="mt-4 text-xs text-center text-muted-foreground">
              Already have an account?{" "}
              <a href="/login" className="underline">
                Log in
              </a>
            </p>
          </CardFooter>
        </Card>
      </main>
    </div>
  );
};
