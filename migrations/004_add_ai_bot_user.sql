INSERT INTO users (id, email, role, name) 
VALUES ('00000000-0000-0000-0000-000000000000', 'ai-bot@system.local', 'agent', 'VoltDesk AI')
ON CONFLICT (email) DO NOTHING;
