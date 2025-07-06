-- Remove is_sign_up column from auth_states table
ALTER TABLE auth_states 
DROP COLUMN is_sign_up;