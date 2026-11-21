-- Match the reference API without rewriting an applied migration or ride data.
alter table public.rides rename column departure_time to departure_date;
alter index public.rides_departure_time_idx rename to rides_departure_date_idx;
