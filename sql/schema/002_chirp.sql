-- +goose Up
CREATE TABLE chirps (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    body text NOT NULL,
    user_id uuid NOT NULL,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS chirps;

