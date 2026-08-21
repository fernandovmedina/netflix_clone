insert into plans(code,name,price,currency,quality,max_streams,active) values
  ('basic','Basic',99.00,'MXN','720p',1,true),
  ('standard','Standard',149.00,'MXN','1080p',2,true),
  ('premium','Premium',219.00,'MXN','4K',4,true)
on conflict(code) do update set name=excluded.name,price=excluded.price,currency=excluded.currency,quality=excluded.quality,max_streams=excluded.max_streams,active=excluded.active;

insert into genres(name) values
 ('Action'),('Adventure'),('Anime'),('Comedy'),('Coming-of-Age'),('Crime'),('Dark Comedy'),('Drama'),
 ('Family'),('Historical Drama'),('Horror'),('Martial Arts'),('Mystery'),('Psychological Thriller'),
 ('Road Movie'),('Romance'),('Satire'),('Sports'),('Teen Drama'),('Thriller'),('War')
on conflict(name) do update set deleted_at=null;

insert into categories(name) values ('Movies'),('Series') on conflict(name) do update set deleted_at=null;
