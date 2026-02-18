-- Migration: 000005 seed default roles
-- Seeds the two base roles: 'user' and 'admin'.
-- INSERT IGNORE prevents failure if already seeded.

INSERT IGNORE INTO roles (name) VALUES ('user'), ('admin');
