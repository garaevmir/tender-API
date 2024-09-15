CREATE TABLE tender (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    service_type VARCHAR(100) NOT NULL,
    status VARCHAR(50) CHECK (status IN ('Created', 'Published', 'Closed')) default 'Created',
    organization_id uuid REFERENCES organization(id) ON DELETE CASCADE,
    creator_username VARCHAR(50) REFERENCES employee(username) ON DELETE CASCADE,
    version INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tender_version (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tender_id UUID REFERENCES tender(id) ON DELETE CASCADE,
    version INT NOT NULL,
    name VARCHAR(100),
    description TEXT,
    service_type VARCHAR(100),
    status VARCHAR(50)
);


CREATE TABLE bid (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    status VARCHAR(50) CHECK (status IN ('Created', 'Published', 'Canceled')) default 'Created',
    tender_id uuid REFERENCES tender(id) ON DELETE CASCADE,
    author_type VARCHAR(50) CHECK (author_type IN ('Organization', 'User')),
    author_id uuid,
    organization_id uuid REFERENCES organization(id) ON DELETE CASCADE,
    version INT DEFAULT 1,
    decision VARCHAR(50) CHECK (decision IN ('Approved', 'Rejected', 'None')) default 'None',
    approved_count int DEFAULT 0, 
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE bid_version (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    bid_id uuid REFERENCES bid(id) ON DELETE CASCADE,
    version INT NOT NULL,
    name VARCHAR(100),
    description TEXT,
    decision VARCHAR(50) CHECK (decision IN ('Approved', 'Rejected', 'None')),
    approved_count int,
    status VARCHAR(50)
);

CREATE TABLE bid_approve (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    bid_id uuid REFERENCES bid(id) ON DELETE CASCADE,
    username VARCHAR(100) REFERENCES employee(username) ON DELETE CASCADE
);

CREATE TABLE bid_review (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    bid_id uuid REFERENCES bid(id) ON DELETE CASCADE,
    username VARCHAR(100) REFERENCES employee(username) ON DELETE cascade,
    review text,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
