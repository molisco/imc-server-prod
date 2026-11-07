-- 0. Limpa o banco caso existam tabelas anteriores (Evita conflitos)
DROP TABLE IF EXISTS stores CASCADE;
DROP TABLE IF EXISTS weekly_master_grids CASCADE;
DROP TABLE IF EXISTS commercials_and_events CASCADE;
DROP TABLE IF EXISTS schedule_style_intervals CASCADE;
DROP TABLE IF EXISTS basic_schedules CASCADE;
DROP TABLE IF EXISTS commercial_block_configs CASCADE;
DROP TABLE IF EXISTS media_library CASCADE;
DROP TABLE IF EXISTS media_styles CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS permission_levels CASCADE;
DROP TABLE IF EXISTS regionals CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;
DROP TYPE IF EXISTS media_type_enum CASCADE;
DROP TYPE IF EXISTS commercial_type_enum CASCADE;

-- Enable the UUID generation extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- 1. ENUMS (Global Enumerations)
-- ============================================================================
CREATE TYPE media_type_enum AS ENUM ('AUDIO', 'VIDEO');
CREATE TYPE commercial_type_enum AS ENUM ('INSTITUTIONAL', 'BLOCK_OPENING', 'BLOCK_CLOSING', 'COMMERCIAL', 'OFFER', 'EVENT');

-- ============================================================================
-- 2. TABLE: TENANTS (Companies / Multi-Tenant Isolation)
-- ============================================================================
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trade_name VARCHAR(150) NOT NULL,
    legal_name VARCHAR(200) NOT NULL,
    document_number VARCHAR(20) UNIQUE NOT NULL,
    
    -- Address
    zip_code VARCHAR(10) NOT NULL,
    street VARCHAR(150) NOT NULL,
    number VARCHAR(20),
    complement VARCHAR(100),
    district VARCHAR(100),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(2) NOT NULL,
    country VARCHAR(100) NOT NULL DEFAULT 'Brasil',
    
    -- Contact Info
    phone VARCHAR(20),
    contact_email VARCHAR(100),
    manager_name VARCHAR(100),
    operational_contacts JSONB DEFAULT '[]'::jsonb,
    financial_contacts JSONB DEFAULT '[]'::jsonb,
    
    -- Visual Identity (White Label)
    logo_url TEXT,
    primary_color VARCHAR(7) NOT NULL DEFAULT '#4364f7',
    background_color VARCHAR(7) NOT NULL DEFAULT '#0f111a',
    font_color VARCHAR(7) NOT NULL DEFAULT '#ffffff',
    field_background_color VARCHAR(7) NOT NULL DEFAULT '#161925',
    
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- 3. TABLE: REGIONALS (Subdivisions of a Tenant)
-- ============================================================================
CREATE TABLE regionals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    coordinator_name VARCHAR(100),
    phone VARCHAR(20),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_regional_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- ============================================================================
-- 4. TABLE: PERMISSION_LEVELS (User Roles)
-- ============================================================================
CREATE TABLE permission_levels (
    id INT PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    description VARCHAR(250)
);

INSERT INTO permission_levels (id, name, description) VALUES
(1, 'Administrator', 'Full system access.'),
(3, 'Operator', 'System operation. Cannot create commercials or users.'),
(5, 'Programmer', 'Can register commercials, no access to create weekly schedules/grids.'),
(7, 'Access Manager', 'Only handles user management and creation.');

-- ============================================================================
-- 5. TABLE: USERS (Portal Backoffice Users)
-- ============================================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID,
    permission_level_id INT NOT NULL,
    name VARCHAR(150) NOT NULL,
    login VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    phone VARCHAR(20),
    job_function VARCHAR(60),
    registration_id VARCHAR(20),
    supervisor_name VARCHAR(100),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_user_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE SET NULL,
    CONSTRAINT fk_user_permission FOREIGN KEY (permission_level_id) REFERENCES permission_levels(id)
);

-- ============================================================================
-- 6. TABLE: MEDIA_STYLES (Music/Video categories)
-- ============================================================================
CREATE TABLE media_styles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    media_type media_type_enum NOT NULL,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(200),
    base_directory VARCHAR(150),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_style_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- ============================================================================
-- 7. TABLE: MEDIA_LIBRARY (Tracks and Video files database)
-- ============================================================================
CREATE TABLE media_library (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    media_style_id UUID NOT NULL,
    media_type media_type_enum NOT NULL,
    display_name VARCHAR(150) NOT NULL,
    file_name VARCHAR(150) NOT NULL,
    duration_seconds INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_media_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_media_style FOREIGN KEY (media_style_id) REFERENCES media_styles(id) ON DELETE RESTRICT
);

-- ============================================================================
-- 8. TABLE: COMMERCIAL_BLOCK_CONFIGS (Interval rules and commercial limits)
-- ============================================================================
CREATE TABLE commercial_block_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    media_type media_type_enum NOT NULL,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(200),
    
    interval_minutes INT NOT NULL DEFAULT 15,
    block_limit_seconds INT NOT NULL DEFAULT 120,
    
    percentage_time_announcement INT NOT NULL DEFAULT 0,
    percentage_temperature INT NOT NULL DEFAULT 0,
    percentage_institutional INT NOT NULL DEFAULT 50,
    percentage_offer INT NOT NULL DEFAULT 50,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_block_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- ============================================================================
-- 9. TABLE: BASIC_SCHEDULES (Time windows for playlists)
-- ============================================================================
CREATE TABLE basic_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    media_type media_type_enum NOT NULL,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(200),
    
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    musical_block_duration_minutes INT NOT NULL DEFAULT 60,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_schedule_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- ============================================================================
-- 10. TABLE: SCHEDULE_STYLE_INTERVALS (Triggers to change styles inside a window)
-- ============================================================================
CREATE TABLE schedule_style_intervals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    basic_schedule_id UUID NOT NULL,
    media_style_id UUID NOT NULL,
    trigger_time TIME NOT NULL,
    
    CONSTRAINT fk_interval_schedule FOREIGN KEY (basic_schedule_id) REFERENCES basic_schedules(id) ON DELETE CASCADE,
    CONSTRAINT fk_interval_style FOREIGN KEY (media_style_id) REFERENCES media_styles(id) ON DELETE RESTRICT
);

-- ============================================================================
-- 11. TABLE: COMMERCIALS_AND_EVENTS (The advertisement tracks)
-- ============================================================================
CREATE TABLE commercials_and_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    media_type media_type_enum NOT NULL,
    commercial_type commercial_type_enum NOT NULL,
    
    name VARCHAR(150) NOT NULL,
    file_name VARCHAR(150) NOT NULL,
    duration_seconds INT NOT NULL DEFAULT 0,
    insertions_per_day INT DEFAULT 0,
    
    -- Validity Date & Time window
    start_date DATE,
    end_date DATE,
    start_time TIME DEFAULT '00:00:00',
    end_time TIME DEFAULT '23:59:59',
    
    run_on_sunday    BOOLEAN NOT NULL DEFAULT TRUE,
    run_on_monday    BOOLEAN NOT NULL DEFAULT TRUE,
    run_on_tuesday   BOOLEAN NOT NULL DEFAULT TRUE,
    run_on_wednesday BOOLEAN NOT NULL DEFAULT TRUE,
    run_on_thursday  BOOLEAN NOT NULL DEFAULT TRUE,
    run_on_friday    BOOLEAN NOT NULL DEFAULT TRUE,
    run_on_saturday  BOOLEAN NOT NULL DEFAULT TRUE,
    
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    event_interruption_mode VARCHAR(20) DEFAULT 'SOFT',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_commercial_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- ============================================================================
-- 12. TABLE: WEEKLY_MASTER_GRIDS (Binds days of the week to schedules/blocks)
-- ============================================================================
CREATE TABLE weekly_master_grids (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    media_type media_type_enum NOT NULL,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(200),
    
    sunday_schedule_id UUID,    sunday_block_id UUID,
    monday_schedule_id UUID,    monday_block_id UUID,
    tuesday_schedule_id UUID,   tuesday_block_id UUID,
    wednesday_schedule_id UUID, wednesday_block_id UUID,
    thursday_schedule_id UUID,  thursday_block_id UUID,
    friday_schedule_id UUID,    friday_block_id UUID,
    saturday_schedule_id UUID,  saturday_block_id UUID,
    holiday_schedule_id UUID,   holiday_block_id UUID,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_weekly_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- ============================================================================
-- 13. TABLE: STORES (The physical edge points executing the Player C#)
-- ============================================================================
CREATE TABLE stores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    regional_id UUID,
    store_name VARCHAR(150) NOT NULL,
    store_number VARCHAR(20),
    
    -- Location fields
    zip_code VARCHAR(10),
    street VARCHAR(150),
    number VARCHAR(20),
    district VARCHAR(100),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(2) NOT NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'America/Sao_Paulo',
    
    -- Local Contact
    phone VARCHAR(20),
    manager_name VARCHAR(100),
    
    -- Media Capability Flags
    has_audio BOOLEAN NOT NULL DEFAULT TRUE,
    has_video BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Live Tracking fields updated by Player sync requests
    last_audio_sync TIMESTAMP WITH TIME ZONE,
    last_video_sync TIMESTAMP WITH TIME ZONE,
    
    token VARCHAR(64) UNIQUE NOT NULL, 
    
    audio_grid_id UUID REFERENCES weekly_master_grids(id) ON DELETE SET NULL,
    video_grid_id UUID REFERENCES weekly_master_grids(id) ON DELETE SET NULL,
    
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_store_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_store_regional FOREIGN KEY (regional_id) REFERENCES regionals(id) ON DELETE CASCADE
);

-- ============================================================================
-- PERFORMANCE & OPERATIONAL INDEXES (ALL TYPOS FIXED)
-- ============================================================================
CREATE INDEX idx_regionals_tenant ON regionals(tenant_id);
CREATE INDEX idx_stores_regional ON stores(regional_id);
CREATE INDEX idx_stores_tenant ON stores(tenant_id);
CREATE UNIQUE INDEX idx_stores_token ON stores(token);

CREATE INDEX idx_library_lookup ON media_library(tenant_id, media_style_id, media_type);
CREATE INDEX idx_commercials_validity ON commercials_and_events(tenant_id, media_type, start_date, end_date, is_active);

-- ============================================================================
-- 14. TABLE: VOICE_PACKS (Plate announcement voice packages)
-- ============================================================================
CREATE TABLE IF NOT EXISTS voice_packs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    base_directory TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- 15. TABLE: CHIMES (Call chimes / signals)
-- ============================================================================
CREATE TABLE IF NOT EXISTS chimes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    duration_seconds INT NOT NULL DEFAULT 0,
    cue_in DOUBLE PRECISION DEFAULT 0.0,
    cue_out DOUBLE PRECISION DEFAULT 0.0,
    lufs DOUBLE PRECISION DEFAULT -14.0,
    amplify DOUBLE PRECISION DEFAULT 1.0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- 16. TABLE: PRE_FORMATTED_WARNINGS (Pre-recorded warning announcements)
-- ============================================================================
CREATE TABLE IF NOT EXISTS pre_formatted_warnings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    duration_seconds INT NOT NULL DEFAULT 0,
    cue_in DOUBLE PRECISION DEFAULT 0.0,
    cue_out DOUBLE PRECISION DEFAULT 0.0,
    lufs DOUBLE PRECISION DEFAULT -14.0,
    amplify DOUBLE PRECISION DEFAULT 1.0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);