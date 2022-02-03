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

