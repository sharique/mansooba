-- Runs once, when the development database volume is first created. Tests get
-- their own databases inside this one, so they never touch development data.
CREATE DATABASE mansooba_test;
