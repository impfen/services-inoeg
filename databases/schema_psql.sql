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
  "name" TEXT,
  "street" TEXT,
  "city" TEXT,
  "zip_code" TEXT,
  "description" TEXT,
  "accessible" BOOL,
  "key_data" TEXT,
  "key_signature" BYTEA,
  "public_key" BYTEA,
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

