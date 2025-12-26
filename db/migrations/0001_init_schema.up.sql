

CREATE TABLE professors (
    id              BIGSERIAL PRIMARY KEY,
    keycloak_id     UUID NOT NULL UNIQUE,
    full_name       TEXT NOT NULL,
    email           TEXT NOT NULL UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE students (
    id              BIGSERIAL PRIMARY KEY,
    keycloak_id     UUID NOT NULL UNIQUE,
    university_id   VARCHAR(32) NOT NULL UNIQUE,
    full_name       TEXT NOT NULL,
    email           TEXT NOT NULL UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE courses (
    id                BIGSERIAL PRIMARY KEY,
    code              VARCHAR(16) NOT NULL UNIQUE,
    title             TEXT NOT NULL,
    term              VARCHAR(16) NOT NULL,
    max_team_size     INTEGER NOT NULL DEFAULT 4,
    add_drop_deadline TIMESTAMPTZ NOT NULL,
    team_lock_date    TIMESTAMPTZ NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (max_team_size > 1),
    CHECK (team_lock_date >= add_drop_deadline)
);

CREATE TABLE course_instructors (
    id           BIGSERIAL PRIMARY KEY,
    course_id    BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    professor_id BIGINT NOT NULL REFERENCES professors(id) ON DELETE CASCADE,
    role         VARCHAR(32) NOT NULL DEFAULT 'primary',
    added_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (course_id, professor_id)
);

CREATE TABLE course_enrollments (
    id          BIGSERIAL PRIMARY KEY,
    course_id   BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    student_id  BIGINT NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    role        VARCHAR(16) NOT NULL DEFAULT 'student',
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (course_id, student_id)
);

CREATE TABLE teams (
    id            BIGSERIAL PRIMARY KEY,
    course_id     BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    formed_by_id  BIGINT NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    status        VARCHAR(16) NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (course_id, name)
);

CREATE TABLE team_memberships (
    id                 BIGSERIAL PRIMARY KEY,
    team_id            BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    student_id         BIGINT NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    added_by_id        BIGINT NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    added_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    requires_approval  BOOLEAN NOT NULL DEFAULT FALSE,
    approved_by_id     BIGINT REFERENCES professors(id) ON DELETE SET NULL,
    approved_at        TIMESTAMPTZ,
    UNIQUE (team_id, student_id)
);

CREATE TABLE team_change_requests (
    id               BIGSERIAL PRIMARY KEY,
    team_id          BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    requested_by_id  BIGINT NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    request_type     VARCHAR(32) NOT NULL,
    payload          JSONB NOT NULL,
    status           VARCHAR(16) NOT NULL DEFAULT 'pending',
    reviewed_by_id   BIGINT REFERENCES professors(id) ON DELETE SET NULL,
    reviewed_at      TIMESTAMPTZ,
    decision_note    TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_course_enrollments_course ON course_enrollments(course_id);
CREATE INDEX idx_course_enrollments_student ON course_enrollments(student_id);
CREATE INDEX idx_teams_course ON teams(course_id);
CREATE INDEX idx_team_memberships_team ON team_memberships(team_id);
CREATE INDEX idx_team_memberships_student ON team_memberships(student_id);
CREATE INDEX idx_team_change_requests_team ON team_change_requests(team_id);
CREATE INDEX idx_course_instructors_course ON course_instructors(course_id);
CREATE INDEX idx_course_instructors_professor ON course_instructors(professor_id);
