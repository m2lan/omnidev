DROP TRIGGER IF EXISTS update_org_members_updated_at ON org_members;
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;
DROP TABLE IF EXISTS org_members CASCADE;
DROP TABLE IF EXISTS organizations CASCADE;
