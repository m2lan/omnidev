DROP TRIGGER IF EXISTS trg_chunk_tsv ON document_chunks;
DROP FUNCTION IF EXISTS update_chunk_tsv();
DROP TRIGGER IF EXISTS update_documents_updated_at ON documents;
DROP TRIGGER IF EXISTS update_knowledge_bases_updated_at ON knowledge_bases;
DROP TABLE IF EXISTS document_chunks CASCADE;
DROP TABLE IF EXISTS documents CASCADE;
DROP TABLE IF EXISTS knowledge_bases CASCADE;
