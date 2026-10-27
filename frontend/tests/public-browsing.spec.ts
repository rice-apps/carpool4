import { expect, test, type Page } from "@playwright/test";

const rideID = "33333333-3333-4333-8333-333333333333";
const viewerID = "44444444-4444-4444-8444-444444444444";
const rice = { id: "11111111-1111-4111-8111-111111111111", title: "Rice" };
const airport = { id: "22222222-2222-4222-8222-222222222222", title: "IAH" };
const owner = {
  id: "55555555-5555-4555-8555-555555555555",
  firstName: "Alice",
  lastName: "Driver",
};
const rider = {
  id: "66666666-6666-4666-8666-666666666666",
  firstName: "Bob",
  lastName: "Rider",
};
const trip = {
  id: rideID,
  departureDate: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
  departureLocation: rice,
  arrivalLocation: airport,
  capacity: 3,
  occupiedSeats: 2,
  status: "RIDE_STATUS_ACTIVE",
};

type RideInput = {
  departureLocationId: string;
  arrivalLocationId: string;
  departureDate: string;
  capacity: number;
  notes: string;
};

async function mockServices(
  page: Page,
  profileExists: { value: boolean },
  scenario: {
    role?: "owner" | "visitor";
    phone?: string;
    state?: "full" | "cancelled" | "past";
  } = {},
) {
  let joinCalls = 0;
  let cancelCalls = 0;
  const created: RideInput[] = [];
  const updated: RideInput[] = [];
  let isOwner = scenario.role === "owner";
  let joined = false;
  let profile = {
    id: viewerID,
    firstName: "Viewer",
    lastName: "Student",
    phone: scenario.phone ?? "7135550123",
  };
  let currentTrip = {
    ...trip,
    capacity: scenario.state === "full" ? 2 : 3,
    status:
      scenario.state === "cancelled"
        ? "RIDE_STATUS_CANCELLED"
        : "RIDE_STATUS_ACTIVE",
    departureDate:
      scenario.state === "past" ? "2020-01-01T12:00:00Z" : trip.departureDate,
  };
  let notes = "Private pickup instructions";

  await page.route("**/mock-auth/**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: "{}",
    });
  });
  await page.route("**/mock-api/**", async (route) => {
    const request = route.request();
    const authenticated =
      request.headers().authorization === "Bearer fixture-token";
    const method = new URL(request.url()).pathname.split("/").at(-1);
    const driver = isOwner ? profile : owner;

    function rideResult() {
      if (!authenticated) return currentTrip;
      return {
        ...currentTrip,
        owner: {
          ...driver,
          ...(isOwner || joined
            ? { phone: "7135550199", email: "driver@rice.edu" }
            : {}),
        },
        riders: [driver, rider, ...(joined ? [profile] : [])],
        notes,
      };
    }

    let result: object;
    let status = 200;
    switch (method) {
      case "ListLocations":
        result = { locations: [rice, airport] };
        break;
      case "ListRides": {
        const input: Partial<RideInput> = request.postDataJSON();
        const matchesFrom =
          !input.departureLocationId ||
          input.departureLocationId === currentTrip.departureLocation.id;
        const matchesTo =
          !input.arrivalLocationId ||
          input.arrivalLocationId === currentTrip.arrivalLocation.id;
        result = {
          rides:
            matchesFrom &&
            matchesTo &&
            currentTrip.status === "RIDE_STATUS_ACTIVE"
              ? [rideResult()]
              : [],
        };
        break;
      }
      case "ListMyRides":
        result = { rides: isOwner || joined ? [rideResult()] : [] };
        break;
      case "GetRide":
        result = { ride: rideResult() };
        break;
      case "GetUser":
        status = profileExists.value ? 200 : 404;
        result = profileExists.value
          ? { user: profile }
          : { code: "not_found", message: "profile not found" };
        break;
      case "UpdateUser": {
        const input: { firstName: string; lastName: string; phone: string } =
          request.postDataJSON();
        profileExists.value = true;
        profile = { ...profile, ...input };
        result = { user: profile };
        break;
      }
      case "CreateRide":
      case "UpdateRide": {
        const input: RideInput = request.postDataJSON();
        (method === "CreateRide" ? created : updated).push(input);
        isOwner = true;
        currentTrip = {
          ...currentTrip,
          departureDate: input.departureDate,
          departureLocation:
            input.departureLocationId === rice.id ? rice : airport,
          arrivalLocation: input.arrivalLocationId === rice.id ? rice : airport,
          capacity: input.capacity,
        };
        notes = input.notes;
        result = { ride: rideResult() };
        break;
      }
      case "JoinRide":
        joinCalls++;
        joined = true;
        currentTrip = {
          ...currentTrip,
          occupiedSeats: currentTrip.occupiedSeats + 1,
        };
        result = { ride: rideResult() };
        break;
      case "LeaveRide":
        joined = false;
        currentTrip = {
          ...currentTrip,
          occupiedSeats: currentTrip.occupiedSeats - 1,
        };
        result = { ride: rideResult() };
        break;
      case "CancelRide":
        cancelCalls++;
        currentTrip = { ...currentTrip, status: "RIDE_STATUS_CANCELLED" };
        result = { ride: rideResult() };
        break;
      default:
        throw new Error(`Unexpected RPC: ${method}`);
    }
    await route.fulfill({
      status,
      contentType: "application/json",
      body: JSON.stringify(result),
    });
  });
  return {
    joinCalls: () => joinCalls,
    cancelCalls: () => cancelCalls,
    created,
    updated,
  };
}

async function signInAsFixture(page: Page) {
  await page.goto("/");
  await page.evaluate(
    ({ id }) => {
      localStorage.setItem(
        "sb-localhost-auth-token",
        JSON.stringify({
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
        }),
      );
    },
    { id: viewerID },
  );
  await page.reload();
}

test("signing out removes cached people and notes", async ({ page }) => {
  await mockServices(page, { value: true });
  await signInAsFixture(page);

  await expect(page.getByText("Alice is driving")).toBeVisible();
  await page.getByRole("link", { name: "View ride from Rice to IAH" }).click();
  await expect(page.getByText("Private pickup instructions")).toBeVisible();
  await expect(
    page.getByRole("heading", { name: /People on this ride/ }),
  ).toBeVisible();

  await page.getByRole("link", { name: "Profile", exact: true }).click();
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/login(?:\?.*)?$/);
  await page
    .getByRole("link", { name: "Browse rides without signing in" })
    .click();
  await expect(page.getByText("Rice student is driving")).toBeVisible();
  await page.getByRole("link", { name: "View ride from Rice to IAH" }).click();
  await expect(page.getByText("Private pickup instructions")).toHaveCount(0);
  await expect(
    page.getByRole("heading", { name: /People on this ride/ }),
  ).toHaveCount(0);
  await expect(page.getByText("Alice Driver")).toHaveCount(0);
});

test("missing profile completes onboarding before returning to the ride", async ({
  page,
}) => {
  const profileExists = { value: false };
  const calls = await mockServices(page, profileExists);
  await signInAsFixture(page);
  await page.goto(`/rides/${rideID}`);

  await expect(page).toHaveURL(`/onboarding?next=%2Frides%2F${rideID}`);
  expect(calls.joinCalls()).toBe(0);
  await page.getByLabel("First name").fill("Viewer");
  await page.getByLabel("Last name").fill("Student");
  await page.getByRole("button", { name: "Finish profile" }).click();

  await expect(page).toHaveURL(`/rides/${rideID}`);
  await expect(
    page.getByRole("button", { name: "Join this ride" }),
  ).toBeVisible();
  expect(calls.joinCalls()).toBe(0);
});

test("guests filter rides, recover from an empty result, and sign in to join", async ({
  page,
}) => {
  await mockServices(page, { value: true });
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto("/");
  await expect(
    page.getByRole("link", { name: "View ride from Rice to IAH" }),
  ).toBeVisible();
  await page.screenshot({
    path: test.info().outputPath("browse-desktop.png"),
    fullPage: true,
  });

  await page.getByLabel("Leaving from").selectOption(airport.id);
  await expect(
    page.getByRole("heading", { name: "No rides found" }),
  ).toBeVisible();
  await page.getByLabel("Going to").selectOption(rice.id);
  await page
    .getByRole("button", { name: "Swap departure and arrival" })
    .click();
  await page.getByRole("link", { name: "View ride from Rice to IAH" }).click();
  await expect(
    page.getByRole("heading", { name: "Driver contact" }),
  ).toHaveCount(0);
  await page.getByRole("link", { name: "Sign in to join" }).click();
  await expect(page).toHaveURL(
    new RegExp("/login\\?next=" + encodeURIComponent("/rides/" + rideID)),
  );
  await expect(
    page.getByRole("button", { name: "Continue with Rice Google" }),
  ).toBeVisible();
});

test("mobile posting validates the route and saves the form fields", async ({
  page,
}) => {
  const calls = await mockServices(page, { value: true });
  await page.setViewportSize({ width: 375, height: 812 });
  await signInAsFixture(page);
  await expect(
    page.getByRole("link", { name: "View ride from Rice to IAH" }),
  ).toBeVisible();
  await page.screenshot({
    path: test.info().outputPath("browse-mobile.png"),
    fullPage: true,
  });
  await page
    .getByRole("navigation")
    .getByRole("link", { name: "Post a ride" })
    .click();
  await expect(
    page.getByRole("heading", { name: "Post a ride", exact: true }),
  ).toBeVisible();
  await page.getByLabel("Leaving from").selectOption(rice.id);
  await page.getByLabel("Going to").selectOption(rice.id);
  await page.getByLabel("Departure date and time").fill("2099-11-20T14:00");
  await page.getByLabel("Total seats, including you").fill("4");
  await page
    .getByLabel("Notes for riders (optional)")
    .fill("Meet at the main gate.");
  await page.getByRole("button", { name: "Post ride", exact: true }).click();
  await expect(page.getByRole("main").getByRole("alert")).toHaveText(
    "Choose different departure and arrival locations.",
  );
  expect(calls.created).toHaveLength(0);

  await page.getByLabel("Going to").selectOption(airport.id);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: test.info().outputPath("post-mobile.png"),
    fullPage: true,
  });
  await page.getByRole("button", { name: "Post ride", exact: true }).click();
  await expect(page).toHaveURL(`/rides/${rideID}`);
  expect(calls.created).toHaveLength(1);
  expect(calls.created[0]).toMatchObject({
    departureLocationId: rice.id,
    arrivalLocationId: airport.id,
    capacity: 4,
    notes: "Meet at the main gate.",
  });
  await expect(page.getByText("Meet at the main gate.")).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Edit ride", exact: true }),
  ).toBeVisible();
});

test("joining shows contact details and allows leaving", async ({ page }) => {
  const calls = await mockServices(page, { value: true });
  await signInAsFixture(page);
  await page.goto(`/rides/${rideID}`);
  await expect(page.getByText("7135550199")).toHaveCount(0);
  await page.getByRole("button", { name: "Join this ride" }).click();
  await expect(page.getByText("You joined this ride.")).toBeVisible();
  await expect(page.getByText("7135550199")).toBeVisible();
  expect(calls.joinCalls()).toBe(1);
  await page.getByRole("button", { name: "Leave ride" }).click();
  await expect(page.getByText("You left this ride.")).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Join this ride" }),
  ).toBeEnabled();
});

test("owners edit rides and must confirm cancellation; history is keyboard accessible", async ({
  page,
}) => {
  const calls = await mockServices(page, { value: true }, { role: "owner" });
  await page.setViewportSize({ width: 375, height: 812 });
  await signInAsFixture(page);
  await page.goto(`/rides/${rideID}`);
  await page.getByRole("link", { name: "Edit ride", exact: true }).click();
  await page
    .getByLabel("Notes for riders (optional)")
    .fill("Updated pickup location.");
  await page.getByRole("button", { name: "Save ride", exact: true }).click();
  await expect(page.getByText("Updated pickup location.")).toBeVisible();
  expect(calls.updated).toHaveLength(1);

  await page.getByRole("button", { name: "Cancel ride", exact: true }).click();
  const dialog = page.getByRole("alertdialog");
  await expect(dialog.getByRole("button", { name: "Keep ride" })).toBeFocused();
  await page.screenshot({ path: test.info().outputPath("cancel-mobile.png") });
  await page.keyboard.press("Escape");
  await expect(dialog).not.toBeVisible();
  expect(calls.cancelCalls()).toBe(0);
  await expect(
    page.getByRole("button", { name: "Cancel ride", exact: true }),
  ).toBeFocused();
  await page.getByRole("button", { name: "Cancel ride", exact: true }).click();
  await dialog
    .getByRole("button", { name: "Cancel ride", exact: true })
    .click();
  await expect(page.getByText("Ride cancelled.")).toBeVisible();
  expect(calls.cancelCalls()).toBe(1);

  await page.getByRole("link", { name: "My rides", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "No upcoming rides" }),
  ).toBeVisible();
  await page.getByRole("tab", { name: "Upcoming", exact: true }).focus();
  await page.keyboard.press("ArrowRight");
  await expect(
    page.getByRole("tab", { name: "Past & cancelled" }),
  ).toHaveAttribute("aria-selected", "true");
  await expect(page.getByRole("tabpanel")).toContainText("cancelled");
  await expect(
    page.getByRole("link", { name: "View ride from Rice to IAH" }),
  ).toBeVisible();
});

test("posting requires a phone number and profile updates unblock it", async ({
  page,
}) => {
  await mockServices(page, { value: true }, { phone: "" });
  await signInAsFixture(page);
  await page.goto("/rides/new");
  await expect(
    page.getByRole("button", { name: "Post ride", exact: true }),
  ).toBeDisabled();
  await page
    .getByRole("navigation")
    .getByRole("link", { name: "Profile", exact: true })
    .click();
  await page.getByLabel("Phone number").fill("7135550123");
  await page.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByRole("status")).toHaveText("Profile saved.");
  await page
    .getByRole("navigation")
    .getByRole("link", { name: "Post a ride" })
    .click();
  await expect(
    page.getByRole("button", { name: "Post ride", exact: true }),
  ).toBeEnabled();
});

for (const state of ["full", "cancelled", "past"] as const) {
  test(`${state} rides cannot be joined`, async ({ page }) => {
    await mockServices(page, { value: true }, { state });
    await signInAsFixture(page);
    await page.goto(`/rides/${rideID}`);
    if (state === "cancelled") {
      await expect(
        page.getByRole("heading", { name: "Trip details" }),
      ).toBeVisible();
      await expect(
        page.getByRole("button", { name: "Join this ride" }),
      ).toHaveCount(0);
    } else {
      const name = state === "full" ? "Ride is full" : "Join this ride";
      await expect(page.getByRole("button", { name })).toBeDisabled();
    }
  });
}

test("a failed ride lookup can be retried", async ({ page }) => {
  await mockServices(page, { value: true });
  await page.route("**/ListRides", async (route) => {
    await route.fulfill({
      status: 503,
      contentType: "application/json",
      body: JSON.stringify({
        code: "unavailable",
        message: "Rides temporarily unavailable",
      }),
    });
  });
  await page.goto("/");
  await expect(page.getByRole("main").getByRole("alert")).toContainText(
    "Couldn’t load rides.",
  );
  await page.unroute("**/ListRides");
  await page.getByRole("button", { name: "Try again" }).click();
  await expect(
    page.getByRole("link", { name: "View ride from Rice to IAH" }),
  ).toBeVisible();
});
