-- The live domain is celestialoud.com (not .co.uk). Update the support address
-- default and any row still holding the old placeholder.
alter table store_settings alter column support_email set default 'hello@celestialoud.com';
update store_settings set support_email = 'hello@celestialoud.com' where support_email = 'hello@celestialoud.co.uk';
alter table store_settings enable row level security;
