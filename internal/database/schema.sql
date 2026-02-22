CREATE TABLE IF NOT EXISTS subnets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cidr TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ips (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subnet_id UUID NOT NULL REFERENCES subnets(id) ON DELETE CASCADE,
    address TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'available', -- available, reserved, allocated
    hostname TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(subnet_id, address)
);
