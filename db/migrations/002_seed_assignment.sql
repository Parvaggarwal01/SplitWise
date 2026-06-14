insert into people (display_name, canonical_key) values
  ('Aisha', 'aisha'),
  ('Rohan', 'rohan'),
  ('Priya', 'priya'),
  ('Meera', 'meera'),
  ('Dev', 'dev'),
  ('Sam', 'sam'),
  ('Kabir', 'kabir')
on conflict (canonical_key) do nothing;

insert into exchange_rates (from_currency, to_currency, rate, effective_on, source) values
  ('USD', 'INR', 83.50, '2026-03-01', 'assignment_fixed_policy')
on conflict do nothing;
