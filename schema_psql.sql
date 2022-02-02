CREATE TABLE "storage" (
  "storage_id" TEXT PRIMARY KEY,
  "data" BYTEA NOT NULL,
  "created_at" TIMESTAMP NOT NULL DEFAULT now(),
  "updated_at" TIMESTAMP NOT NULL DEFAULT now(),
  "accessed_at" TIMESTAMP NOT NULL DEFAULT now()
);

