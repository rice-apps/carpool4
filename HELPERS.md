# Carpool4 provided helpers

## RPC helpers

### Require sign-in: requireSignedInUser(ctx)

Gets the signed-in person's user ID and email from ctx and returns them as app.CurrentUser. This identifies the person making the request.

The identity in ctx has already been checked by the sign-in code. This helper reads that identity; it does not sign someone in or load their saved profile.

**Inputs and outputs**

`func requireSignedInUser(ctx context.Context) (app.CurrentUser, error)`

**Example**

```go
currentUser, err := requireSignedInUser(ctx)
if err != nil {
    return nil, convertToConnectError(err)
}
```

The person asking and the profile being requested can be different. If Mia opens Alex's profile, currentUser describes Mia, while the profile ID in the request identifies Alex. Saving a profile uses the current user's checked ID and email.

- ctx contains the checked identity associated with this request.

- currentUser has UserID and Email from sign-in. Names and phone numbers come from the person's saved profile instead.

- Missing identity or an empty ID returns app.ErrUnauthenticated.

### Allow guests: auth.GetSignedInUser(ctx)

Checks whether ctx contains a signed-in identity. GetRide and ListRides allow guests, so the absence of an identity is an allowed case for those requests.

Returns the auth.User from ctx and a boolean saying whether one was present. ListLocations needs no identity lookup; other RPC operations require sign-in.

**Inputs and outputs**

`func GetSignedInUser(ctx context.Context) (User, bool)`

**Example**

```go
currentUser := app.CurrentUser{}
if _, signedIn := auth.GetSignedInUser(ctx); signedIn {
    var err error
    currentUser, err = requireSignedInUser(ctx)
    if err != nil {
        return nil, convertToConnectError(err)
    }
}
```

An empty app.CurrentUser{} represents a guest. A signed-in request has a CurrentUser containing that person's checked ID and email.

- auth.User contains the ID and email checked by the sign-in code. app.CurrentUser contains the identity in the form expected by the application functions.

- signedIn is false when no identity was supplied. The request can then be handled as a guest request.

- An invalid sign-in token is rejected before the handler runs. It does not become a guest request.

### Return an API error: convertToConnectError(err)

Converts a server-side error into the Connect error code and message sent to the website.

For example, a missing ride should be reported as NotFound. An unexpected database failure should receive a general message, while the details remain available in the server logs for debugging.

**Inputs and outputs**

`func convertToConnectError(err error) error`

**Example**

```go
return nil, convertToConnectError(err)
```

- Nil is preserved.

- The helper recognizes errors used by the application. For example, app.ErrInvalidArgument becomes InvalidArgument, meaning the supplied information was not acceptable. app.ErrRideNotFound becomes NotFound, meaning the requested ride does not exist.

- An error the helper does not recognize becomes Internal, meaning an unexpected server problem. The response uses a general message so private database details are not sent to the website. The original failure is still available to the server.

### Convert a user result: convertUserToProto(user)

Copies an app.User result into carpoolv1.User for the API response. The input is the profile already prepared for the person asking.

**Inputs and outputs**

`func convertUserToProto(user app.User) *carpoolv1.User`

**Example**

```go
response := &carpoolv1.GetUserResponse{User: convertUserToProto(user)}
return connect.NewResponse(response), nil
```

user is the profile returned by the application, including only the contacts this person may see. response contains that profile in the API's format.

This helper copies ID, names, email, and phone from the app.User it receives. If the application left email and phone empty because this person cannot see them, they stay empty in the response. The permission decision must happen in app before you call this helper.

### Convert a ride result: convertRideToProto(ride)

Copies an app.Ride result into carpoolv1.Ride for the API response. It also converts the people, locations, time, and status inside the ride.

**Inputs and outputs**

`func convertRideToProto(ride app.Ride) *carpoolv1.Ride`

**Example**

```go
ride, err := s.service.GetRide(ctx, currentUser, rideID)
if err != nil {
    return nil, convertToConnectError(err)
}
response := &carpoolv1.GetRideResponse{Ride: convertRideToProto(ride)}
return connect.NewResponse(response), nil
```

- The ride result includes its ID, departure time, starting place, destination, total seats, occupied seats, and whether it is active or cancelled. People and notes are copied when the application included them.

- For a guest, the application leaves out the owner, riders, and notes. Those fields stay absent or empty in the API response. The occupied-seat count is still included, even though the list of people is hidden.

- This helper changes the data's format. The privacy helpers in app decide which fields a person may receive before this conversion happens.

### Convert a location: convertLocationToProto(location)

Copies one app.LocationRecord into the API's location format. The location represents a pickup point or destination.

**Inputs and outputs**

`func convertLocationToProto(location app.LocationRecord) *carpoolv1.Location`

**Example**

```go
locations[i] = convertLocationToProto(location)
```

location is one entry from the application's locations list. locations contains the converted entries for the response.

ID, Title, and Address are copied. The helper does not fetch locations from the database.

## Application helpers

A viewer is the person receiving the information: the current user or a guest. A result is the information prepared to send back to that person.

### What the user and ride values mean

**CurrentUser** describes the person making the request. It has their UserID and Email from checked sign-in information. It does not contain their full saved profile. If Mia opens Alex's profile, CurrentUser describes Mia. An empty CurrentUser{} represents a guest.

**UserRecord** describes a profile read from the database. It includes the ID, first name, last name, email, and phone number that are stored there. In Mia's request to open Alex's profile, the UserRecord describes Alex. Having Alex's phone number in this record does not mean Mia is allowed to receive it.

**User** describes the profile information you will return to the person who asked. It keeps the ID and names, and includes email and phone only when that person may see them. If Mia cannot see Alex's contacts, the User for Alex has empty Email and Phone fields. The UserRecord still contains whatever contacts were saved.

**LoadedRide** contains the information read from the database about a ride, including its owner's full UserRecord, its riders' UserRecord values, locations, time, notes, and seat capacity. Loaded means the information has been fetched. It still includes stored contacts, so it needs to be prepared for the person asking before you return it.

**Ride** is the ride information prepared for that person. For a signed-in user, it includes people with only the contacts they may see. For a guest, it includes basic trip information and occupied seats, with no owner, riders, or notes.

The privacy rule is that a signed-in person may see their own contacts and those of people with whom they share any ride. Past and cancelled rides count too. Hiding contacts means leaving them out of the returned copy; it does not erase them from the database.

### Require sign-in: requireSignIn(currentUser)

Checks that currentUser has a user ID. An empty ID returns ErrUnauthenticated; a present ID returns nil.

requireSignedInUser gets the checked identity from ctx. requireSignIn checks the CurrentUser it was given.

**Inputs and outputs**

`func requireSignIn(currentUser CurrentUser) error`

**Example**

```go
if err := requireSignIn(currentUser); err != nil {
    return Ride{}, err
}
```

This check only establishes that a user ID is present. It does not check ownership, available seats, or other rules for the requested action.

GetRide and ListRides allow an empty CurrentUser for guests. ListLocations has no current-user argument.

### Get a visible profile: getVisibleProfile(ctx, tx, currentUser, record)

Returns a User result from a stored UserRecord. It always includes the ID and names, and decides whether to include email and phone using the identity of the person asking.

Pass currentUser, the person making the request, and record, the profile you already loaded. The helper checks the contact privacy rule for you. Use the same tx that read or saved the profile.

**Inputs and outputs**

`func getVisibleProfile(ctx context.Context, tx Transaction, currentUser CurrentUser, record UserRecord) (User, error)`

**Example**

```go
profile, err := getVisibleProfile(ctx, tx, currentUser, record)
if err != nil {
    return User{}, err
}
```

record is the saved profile, currentUser identifies the person asking, and profile is the result prepared for that person. If Mia opens Alex Chen's profile, the helper includes Alex's contacts only when Mia shares a ride with Alex. Otherwise, it returns Alex's ID and names with empty contact fields.

The helper leaves record unchanged. An empty phone number in the response does not delete Alex's saved phone number. When contacts are allowed, it copies whatever was saved; a missing phone number stays empty.

- ID, FirstName, and LastName are always copied.

- Email and Phone are included for the current user's own profile or someone with whom they share a ride. Past and cancelled rides count too. Otherwise, both fields are empty.

- Guests receive no contacts. A failed permission lookup returns an error and no profile. Own-profile and guest results do not need a permission lookup.

### Build ride results: buildRidesForViewer(ctx, tx, viewerID, rides)

Builds a list of Ride results from a list of LoadedRide values. It checks which people's contacts the viewer may see and prepares each person's profile using that decision.

For example, Mia may be allowed to see Alex's contacts but not Jordan's. A ride result can include both people's names while showing only Alex's email and phone. The permission is decided separately for each person.

**Inputs and outputs**

`func buildRidesForViewer(ctx context.Context, tx Transaction, viewerID uuid.UUID, rides []LoadedRide) ([]Ride, error)`

**Example**

```go
results, err := buildRidesForViewer(ctx, tx, currentUser.UserID, loadedRides)
if err != nil {
    return nil, err
}
```

loadedRides contains the rides read from the database. currentUser.UserID identifies the person asking, and results contains the rides prepared for that person. The helper checks contact permission for all owners and riders in one database call, then prepares their profiles with only the allowed contacts.

tx is the same transaction that loaded the rides. The ride data and contact decisions therefore use a consistent view of the database.

- ctx carries cancellation and the request's time limit. The helper uses the existing tx rather than opening another transaction.

- viewerID identifies the person receiving the result. uuid.Nil represents a guest; an empty CurrentUser has that value in UserID.

- The returned rides stay in the same order as the input. A failed contact permission lookup returns an error.

- Guests get the route, time, seats, and status, with no owner, riders, or notes. Because those people are not included, a guest needs no contact permission query.

- Signed-in people get the names and notes, with contacts only where allowed by the shared-ride rule. OccupiedSeats counts everyone in the ride, including the driver. Hiding the people from a guest does not change that count.

## Storage access

### Start a transaction: s.database.BeginTx(ctx)

Starts a transaction and returns it as tx. err reports whether starting it failed. The operation's database reads, writes, and contact checks all use this same tx.

**Inputs and outputs**

`BeginTx(ctx context.Context) (Transaction, error)`

**Example**

```go
tx, err := s.database.BeginTx(ctx)
if err != nil {
    return Ride{}, err
}
defer tx.Rollback()
```

ctx supplies the request's cancellation and time limit. BeginTx uses Serializable isolation.

### Get profile methods: tx.Users()

Returns the profile repository for tx. Its methods find or save profiles and check contact permission. Getting the repository does not load any profiles.

**Inputs and outputs**

`Users() UserRepository`

**Example**

```go
users := tx.Users()
```

users provides profile access within tx. It is not a list of user profiles.

### Get ride methods: tx.Rides()

Returns the ride repository for tx. Its methods read, list, create, update, or cancel rides and change who is in them. Getting the repository runs no query.

**Inputs and outputs**

`Rides() RideRepository`

**Example**

```go
rides := tx.Rides()
```

rides provides ride access within tx. It is not a list of loaded rides.

### Get location methods: tx.Locations()

Returns the location repository for tx. Its ListLocations method supplies the pickup and destination choices.

**Inputs and outputs**

`Locations() LocationRepository`

**Example**

```go
locations := tx.Locations()
```

locations provides location access within tx. Getting the repository does not fetch the locations list.

### Undo unfinished changes: tx.Rollback()

Ends tx without keeping uncommitted changes.

For example, if adding a person to a ride succeeds but preparing the response fails, rollback undoes the addition. The database is not left with a partially successful operation.

**Inputs and outputs**

`Rollback() error`

**Example**

```go
defer tx.Rollback()
```

For a read-only operation, rollback closes tx. It does not delete stored information or erase the results already read.

- Returns nil after a successful rollback or when tx is already closed.

- Rollback is safe after a successful commit. It does not undo committed changes.

### Keep your database changes: tx.Commit()

Finishes tx and keeps its changes. A failed commit returns an error. Database conflict errors are already translated into app.ErrTransactionConflict.

The returned result, including its contact privacy decisions, is prepared before commit. If that preparation fails, the changes can still be rolled back.

**Inputs and outputs**

`Commit() error`

**Example**

```go
if err := tx.Commit(); err != nil {
    return Ride{}, err
}
return result, nil
```

May return ErrTransactionConflict when simultaneous operations prevent the transaction from completing consistently. There is no automatic retry. A retry repeats the whole operation in a new transaction, including its reads and checks.

### Check profile contacts: ListContactVisibleUserIDs

Returns the requested profile IDs whose contacts the viewer may see. The inputs identify the viewer and the profiles being checked.

If Mia asks about Alex and Jordan but only shares a ride with Alex, visibleIDs contains Alex's ID and leaves out Jordan's. The answer contains IDs, not profiles or contact details. getVisibleProfile and buildRidesForViewer use this lookup to check contact permission.

**Inputs and outputs**

`ListContactVisibleUserIDs(ctx context.Context, viewerID uuid.UUID, targetIDs []uuid.UUID) ([]uuid.UUID, error)`

**Example**

```go
visibleIDs, err := tx.Users().ListContactVisibleUserIDs(
    ctx, currentUser.UserID, []uuid.UUID{record.ID},
)
if err != nil {
    return nil, err
}
// The returned IDs identify the profiles whose contacts are allowed.
return visibleIDs, nil
```

visibleIDs contains the allowed profile IDs. You do not need to turn this list into a contact-permission flag when returning a profile: getVisibleProfile handles the check and builds the profile result. buildRidesForViewer handles it for ride results.

- viewerID identifies the person receiving the information. targetIDs contains the IDs of the people whose contacts are being checked.

- An empty returned list with no error means none of the requested contacts are allowed.

- A person can see their own contacts and those of anyone with whom they share a ride. The shared ride may be past or cancelled; it does not have to be the ride currently being viewed.

- An error means the permission check could not be completed. That is different from a successful check returning no allowed IDs.

## Postgres helpers

### Convert a user row: convertUserRow(row)

Copies a sqlc.User database row into an app.UserRecord, retaining the stored profile information.

**Inputs and outputs**

`func convertUserRow(user sqlc.User) app.UserRecord`

**Example**

```go
record := convertUserRow(row)
```

row contains the profile returned by the database. record is that profile in the application's stored-record format.

The helper copies ID, FirstName, LastName, Email, and Phone. It includes the full stored contacts. For a profile response, getVisibleProfile checks permission and prepares the profile for the person who asked.

### Convert a location row: convertLocationRow(row)

Copies a sqlc.Location database row into an app.LocationRecord with the same ID, title, and address.

**Inputs and outputs**

`func convertLocationRow(location sqlc.Location) app.LocationRecord`

**Example**

```go
location := convertLocationRow(row)
```

row contains the database location. location is the corresponding application record.

The location's ID, Title, and Address are copied. Locations have no personal contact fields, so they do not need the user contact privacy step.

### Assemble a stored ride: convertRideRows(...)

Combines the ride, owner's profile, starting location, destination, and riders into one app.LoadedRide.

The rider information arrives as JSON. The helper decodes it into rider profiles and includes them in the loaded ride.

**Inputs and outputs**

`func convertRideRows(ride sqlc.Ride, owner sqlc.User, departure, arrival sqlc.Location, ridersJSON json.RawMessage) (app.LoadedRide, error)`

**Example**

```go
loaded, err := convertRideRows(row.Ride, row.User, row.Location, row.Location_2, row.Riders)
```

row contains the pieces returned by the ride query. loaded is the assembled LoadedRide. err reports whether the rider JSON could be decoded.

- ride contains the trip details; owner is the driver's profile; departure and arrival are the starting place and destination; ridersJSON contains the encoded rider profiles.

- In this example, row.User is the owner, row.Location is the departure location, and row.Location_2 is the arrival location. Exact field names depend on the query.

- Invalid rider JSON returns an error.

- **LoadedRide** retains the owner's and riders' stored contacts. Contact privacy is applied when the application builds the Ride result for a viewer.

### Translate a user query error: translateUserError(err)

Translates sql.ErrNoRows into an error recognizable as app.ErrUserNotFound, meaning the requested profile was not found.

**Inputs and outputs**

`func translateUserError(err error) error`

**Example**

```go
if err != nil {
    return app.UserRecord{}, translateUserError(err)
}
return convertUserRow(row), nil
```

RPC reports this as NotFound to the website. The original database error remains attached for debugging.

A database conflict between simultaneous transactions becomes app.ErrTransactionConflict. Passing nil returns nil. Errors this helper does not recognize are returned unchanged rather than hidden or treated as success.

### Translate a ride query error: translateRideError(err)

Translates sql.ErrNoRows into an error recognizable as app.ErrRideNotFound, meaning the requested ride was not found.

**Inputs and outputs**

`func translateRideError(err error) error`

**Example**

```go
return app.LoadedRide{}, translateRideError(err)
```

An empty ride list can be a successful answer. It does not by itself mean that a particular requested ride was missing.

Also translates transaction conflicts and preserves unknown errors. Nil is preserved.

### Translate a ride write error: translateRideWriteError(err)

Recognizes database failures caused by a missing related item or a failed rule on the ride data.

**Inputs and outputs**

`func translateRideWriteError(err error) error`

**Example**

```go
return translateRideWriteError(err)
```

Foreign-key and check-constraint failures become app.ErrInvalidRideWrite. For example, a ride may refer to a location that does not exist. RPC reports this error as InvalidArgument.

- A conflict between simultaneous transactions becomes app.ErrTransactionConflict. This helper reports the failure; it does not retry or fix the data.

- Nil is preserved. Unrecognized database errors are returned unchanged.

- The original database failure remains attached to the application error.

### Translate other query errors: translateTransactionError(err)

Recognizes database transaction conflicts and returns an error identifiable as app.ErrTransactionConflict.

**Inputs and outputs**

`func translateTransactionError(err error) error`

**Example**

```go
return translateTransactionError(err)
```

Database error code 40001 becomes app.ErrTransactionConflict. Other errors are preserved, including nil.
