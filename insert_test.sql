-- 1. Inserir um Tenant (Empresa fictícia de varejo)
INSERT INTO tenants (id, trade_name, legal_name, document_number, zip_code, street, number, city, state)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', -- UUID fixo para nosso teste
    'Supermercados Neomarket',
    'Neomarket Comercio de Alimentos LTDA',
    '12.345.678/0001-99',
    '80000-000',
    'Avenida das Araucárias',
    '1500',
    'Curitiba',
    'PR'
);

-- 2. Inserir uma Regional vinculada a esse Tenant
INSERT INTO regionals (id, tenant_id, name, coordinator_name)
VALUES (
    'b1111111-9c0b-4ef8-bb6d-6bb9bd380a22',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', -- Aponta para o Tenant acima
    'Região Metropolitana de Curitiba',
    'Carlos Coordenador'
);

-- 3. Inserir a Loja de Testes com o Token que usaremos na API
INSERT INTO stores (id, tenant_id, regional_id, store_name, store_number, city, state, token, has_audio, has_video)
VALUES (
    'c2222222-9c0b-4ef8-bb6d-6bb9bd380a33',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', -- Aponta para o Tenant
    'b1111111-9c0b-4ef8-bb6d-6bb9bd380a22', -- Aponta para a Regional
    'Neomarket - Loja Centro',
    'Loja #101',
    'Curitiba',
    'PR',
    'TOKEN_PLAYER_TESTE_SHA256_ABC123', -- Esse é o Token secreto da loja!
    true,
    false
);