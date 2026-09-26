import test from "node:test";
import assert from "node:assert/strict";
import { create } from "@bufbuild/protobuf";
import { RideSchema } from "../gen/carpool/v1/ride_pb";
import { availableSeats } from "./ride";

test("full guest ride has no available seats", () => {
  const ride = create(RideSchema, { capacity: 2, occupiedSeats: 2, riders: [] });
  assert.equal(availableSeats(ride), 0);
});
