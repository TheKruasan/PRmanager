 CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
 
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(50) PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
 
CREATE TABLE IF NOT EXISTS pull_requests (
    id VARCHAR(50) PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    author_id VARCHAR(50) NOT NULL REFERENCES users(id),
    status VARCHAR(20) DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'MERGED')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    merged_at TIMESTAMP WITH TIME ZONE NULL
);
 
CREATE TABLE IF NOT EXISTS pr_reviewers (
    pr_id VARCHAR(50) NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    user_id VARCHAR(50) NOT NULL REFERENCES users(id),
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (pr_id, user_id)
);
 
CREATE INDEX IF NOT EXISTS idx_users_team_id ON users(team_id);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
CREATE INDEX IF NOT EXISTS idx_pull_requests_author_id ON pull_requests(author_id);
CREATE INDEX IF NOT EXISTS idx_pull_requests_status ON pull_requests(status);
CREATE INDEX IF NOT EXISTS idx_pr_reviewers_user_id ON pr_reviewers(user_id);
CREATE INDEX IF NOT EXISTS idx_teams_name ON teams(name);