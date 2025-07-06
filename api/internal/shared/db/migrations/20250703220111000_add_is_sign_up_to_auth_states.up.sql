-- Add is_sign_up column to auth_states table to distinguish between sign-up and login flows
ALTER TABLE auth_states 
ADD COLUMN is_sign_up boolean NOT NULL DEFAULT false;