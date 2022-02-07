CREATE TABLE "appointment" (
  "appointment_id" TEXT PRIMARY KEY,
  "provider" TEXT REFERENCES "provider",
  "free_slots" INT DEFAULT 0,
  "duration" INT NOT NULL,
  "timestamp" TIMESTAMPTZ NOT NULL,
  "vaccine" TEXT NOT NULL,
  "signed_data" TEXT,
  "signature" BYTEA,
  "public_key" BYTEA,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE "slot" (
  "slot_id" TEXT PRIMARY KEY,
  "appointment" TEXT NOT NULL REFERENCES "appointment",
  "token" TEXT,
  "public_key" BYTEA,
  "encrypted_data" JSONB,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE "property" (
  "key" TEXT,
  "value" TEXT,
  "appointment" TEXT REFERENCES "appointment",
  PRIMARY KEY ("key", "appointment")
);
CREATE INDEX ON "property" ("key", "value");

CREATE TABLE "user_token" (
  "user_id" TEXT PRIMARY KEY,
  "n" INT NOT NULL DEFAULT 0
);

CREATE TABLE "used_token" (
  "token_id" TEXT PRIMARY KEY
);

CREATE TABLE "token" (
  "name" TEXT PRIMARY KEY,
  "n" INT NOT NULL DEFAULT 0
);
INSERT INTO "token" ("name") VALUES ('primary') ON CONFLICT DO NOTHING;

CREATE TABLE "mediator" (
  "mediator_id" TEXT PRIMARY KEY,
  "key_data" TEXT NOT NULL,
  "key_signature" BYTEA NOT NULL,
  "public_key" BYTEA NOT NULL,
  "active" BOOL NOT NULL DEFAULT true,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE "provider" (
  "provider_id" TEXT PRIMARY KEY,
  "name" TEXT NOT NULL DEFAULT '',
  "street" TEXT NOT NULL DEFAULT '',
  "city" TEXT NOT NULL DEFAULT '',
  "zip_code" TEXT NOT NULL DEFAULT '',
  "description" TEXT NOT NULL DEFAULT '',
  "accessible" BOOL NOT NULL DEFAULT false,
  "key_data" TEXT NOT NULL DEFAULT '',
  "key_signature" BYTEA NOT NULL DEFAULT '',
  "public_key" BYTEA NOT NULL DEFAULT '',
  "active" BOOL NOT NULL DEFAULT false,
  "unverified_data" JSONB,
  "verified_data" JSONB,
  "confirmed_data" JSONB,
  "public_data" JSONB,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE "storage" (
  "storage_id" TEXT PRIMARY KEY,
  "data" BYTEA NOT NULL,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "accessed_at" TIMESTAMPTZ NOT NULL DEFAULT now()
);

