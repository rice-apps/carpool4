insert into locations (title, address) values
  ('Rice University', '6100 Main St, Houston, TX 77005'),
  ('George Bush Intercontinental Airport (IAH)', '2800 N Terminal Rd, Houston, TX 77032'),
  ('William P. Hobby Airport (HOU)', '7800 Airport Blvd, Houston, TX 77061')
on conflict (address) do nothing;
