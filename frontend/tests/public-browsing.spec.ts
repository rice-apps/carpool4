import { expect, test, type Page } from "@playwright/test";

const rideID = "33333333-3333-4333-8333-333333333333";
const viewerID = "44444444-4444-4444-8444-444444444444";
const rice = { id: "11111111-1111-4111-8111-111111111111", title: "Rice" };
const airport = { id: "22222222-2222-4222-8222-222222222222", title: "IAH" };
const owner = { id: "55555555-5555-4555-8555-555555555555", firstName: "Alice", lastName: "Driver" };
const rider = { id: "66666666-6666-4666-8666-666666666666", firstName: "Bob", lastName: "Rider" };
const trip = {
  id: rideID,
  departureTime: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
  departureLocation: rice,
  arrivalLocation: airport,
  capacity: 3,
  occupiedSeats: 2,
  status: "RIDE_STATUS_ACTIVE",
};

async function mockServices(page: Page, profileExists: { value: boolean }) {
  let joinCalls = 0;
  await page.route("**/mock-auth/**", async (route) => {
    await route.fulfill({ status: 200, contentType: "application/json", body: "{}" });
  });
  await page.route("**/mock-api/**", async (route) => {
    const request = route.request();
    const authenticated = request.headers().authorization === "Bearer fixture-token";
    const ride = authenticated
      ? { ...trip, owner, riders: [owner, rider], notes: "Private pickup instructions" }
      : trip;
    const method = new URL(request.url()).pathname.split("/").at(-1);
    let result: object;
    let status = 200;
    switch (method) {
      case "ListLocations":
        result = { locations: [rice, airport] };
        break;
      case "ListRides":
        result = { rides: [ride] };
        break;
      case "GetRide":
        result = { ride };
        break;
      case "GetUser":
        status = profileExists.value ? 200 : 404;
        result = profileExists.value
          ? { user: { id: viewerID, firstName: "Viewer", lastName: "Student" } }
          : { code: "not_found", message: "profile not found" };
        break;
      case "UpdateUser":
        profileExists.value = true;
        result = { user: { id: viewerID, firstName: "Viewer", lastName: "Student" } };
        break;
      case "JoinRide":
        joinCalls++;
        result = { ride };
        break;
      default:
        throw new Error(`Unexpected RPC: ${method}`);
    }
    await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(result) });
  });
  return { joinCalls: () => joinCalls };
}

async function signInAsFixture(page: Page) {
  await page.goto("/");
  await page.evaluate(({ id }) => {
    localStorage.setItem("sb-localhost-auth-token", JSON.stringify({
      access_token: "fixture-token",
      refresh_token: "fixture-refresh",
      token_type: "bearer",
      expires_at: Math.floor(Date.now() / 1000) + 3600,
      expires_in: 3600,
      user: {
        id,
        aud: "authenticated",
        role: "authenticated",
        email: "viewer@rice.edu",
        app_metadata: {},
        user_metadata: {},
        created_at: "2026-09-25T00:00:00Z",
      },
    }));
  }, { id: viewerID });
  await page.reload();
}

test("signing out removes cached people and notes", async ({ page }) => {
  await mockServices(page, { value: true });
  await signInAsFixture(page);

  await expect(page.getByText("Alice is driving")).toBeVisible();
  await page.getByRole("link", { name: /Rice IAH/ }).click();
  await expect(page.getByText("Private pickup instructions")).toBeVisible();
  await expect(page.getByRole("heading", { name: /People on this ride/ })).toBeVisible();

  await page.getByRole("link", { name: "Your profile" }).click();
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/login(?:\?.*)?$/);
  await page.getByRole("link", { name: "Browse rides without signing in" }).click();
  await expect(page.getByText("Rice student is driving")).toBeVisible();
  await page.getByRole("link", { name: /Rice IAH/ }).click();
  await expect(page.getByText("Private pickup instructions")).toHaveCount(0);
  await expect(page.getByRole("heading", { name: /People on this ride/ })).toHaveCount(0);
  await expect(page.getByText("Alice Driver")).toHaveCount(0);
});

test("missing profile completes onboarding before returning to the ride", async ({ page }) => {
  const profileExists = { value: false };
  const calls = await mockServices(page, profileExists);
  await signInAsFixture(page);
  await page.goto(`/rides/${rideID}`);

  await expect(page).toHaveURL(`/onboarding?next=%2Frides%2F${rideID}`);
  expect(calls.joinCalls()).toBe(0);
  await page.getByPlaceholder("Your first name").fill("Viewer");
  await page.getByPlaceholder("Your last name").fill("Student");
  await page.getByRole("button", { name: /Finish profile/ }).click();

  await expect(page).toHaveURL(`/rides/${rideID}`);
  await expect(page.getByRole("button", { name: /Join this ride/ })).toBeVisible();
  expect(calls.joinCalls()).toBe(0);
});
