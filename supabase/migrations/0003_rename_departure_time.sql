-- Rename back to departure_time: the column holds a full timestamp, not a date.
alter table public.rides rename column departure_date to departure_time;
alter index public.rides_departure_date_idx rename to rides_departure_time_idx;
