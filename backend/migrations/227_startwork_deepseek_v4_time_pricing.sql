-- Configure official DeepSeek V4 valley prices with Beijing peak multipliers
-- for Startwork-managed organization groups.

CREATE TEMP TABLE startwork_deepseek_target_groups ON COMMIT DROP AS
SELECT
    g.id AS group_id,
    CASE
        WHEN LOWER(BTRIM(g.platform)) IN ('', 'openai_chat') THEN 'openai'
        ELSE LOWER(BTRIM(g.platform))
    END AS pricing_platform
FROM groups g
WHERE g.deleted_at IS NULL
  AND (
      g.name LIKE 'deepseek-ecosystem-%'
      OR g.name LIKE 'deepseek-private-%'
  );

-- Group pricing has higher precedence than channel pricing. Remove only the
-- DeepSeek entries so the channel time multiplier can take effect.
WITH rebuilt AS (
    SELECT
        g.id,
        COALESCE(
            jsonb_agg(jsonb_set(entry.value, '{models}', filtered.models))
                FILTER (WHERE jsonb_array_length(filtered.models) > 0),
            '[]'::jsonb
        ) AS model_pricing
    FROM groups g
    JOIN startwork_deepseek_target_groups target ON target.group_id = g.id
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(g.model_pricing, '[]'::jsonb)) AS entry(value)
    CROSS JOIN LATERAL (
        SELECT COALESCE(jsonb_agg(to_jsonb(model_name)), '[]'::jsonb) AS models
        FROM jsonb_array_elements_text(COALESCE(entry.value->'models', '[]'::jsonb)) AS model_name
        WHERE LOWER(BTRIM(model_name)) NOT IN (
            'deepseek-v4-flash',
            'deepseek-v4-pro',
            'deepseek-chat',
            'deepseek-reasoner'
        )
    ) filtered
    GROUP BY g.id
)
UPDATE groups g
SET model_pricing = rebuilt.model_pricing,
    updated_at = NOW()
FROM rebuilt
WHERE g.id = rebuilt.id
  AND g.model_pricing IS DISTINCT FROM rebuilt.model_pricing;

INSERT INTO channels (name, description, status)
SELECT
    'Startwork DeepSeek 峰谷价 ' || target.group_id,
    'Startwork managed DeepSeek group using Asia/Shanghai valley prices and 2x peak periods',
    'active'
FROM startwork_deepseek_target_groups target
WHERE NOT EXISTS (
    SELECT 1
    FROM channel_groups cg
    WHERE cg.group_id = target.group_id
)
ON CONFLICT (name) DO NOTHING;

INSERT INTO channel_groups (channel_id, group_id)
SELECT c.id, target.group_id
FROM startwork_deepseek_target_groups target
JOIN channels c ON c.name = 'Startwork DeepSeek 峰谷价 ' || target.group_id
WHERE NOT EXISTS (
    SELECT 1
    FROM channel_groups cg
    WHERE cg.group_id = target.group_id
)
ON CONFLICT (group_id) DO NOTHING;

CREATE TEMP TABLE startwork_deepseek_target_channels ON COMMIT DROP AS
SELECT DISTINCT cg.channel_id, target.pricing_platform
FROM startwork_deepseek_target_groups target
JOIN channel_groups cg ON cg.group_id = target.group_id;

WITH rewritten AS (
    SELECT
        cmp.id,
        COALESCE(
            jsonb_agg(to_jsonb(model_name))
                FILTER (WHERE LOWER(BTRIM(model_name)) NOT IN (
                    'deepseek-v4-flash',
                    'deepseek-v4-pro',
                    'deepseek-chat',
                    'deepseek-reasoner'
                )),
            '[]'::jsonb
        ) AS models
    FROM channel_model_pricing cmp
    JOIN startwork_deepseek_target_channels target
      ON target.channel_id = cmp.channel_id
     AND target.pricing_platform = LOWER(BTRIM(cmp.platform))
    CROSS JOIN LATERAL jsonb_array_elements_text(cmp.models) AS model_name
    GROUP BY cmp.id
)
DELETE FROM channel_model_pricing cmp
USING rewritten
WHERE cmp.id = rewritten.id
  AND jsonb_array_length(rewritten.models) = 0;

WITH rewritten AS (
    SELECT
        cmp.id,
        COALESCE(
            jsonb_agg(to_jsonb(model_name))
                FILTER (WHERE LOWER(BTRIM(model_name)) NOT IN (
                    'deepseek-v4-flash',
                    'deepseek-v4-pro',
                    'deepseek-chat',
                    'deepseek-reasoner'
                )),
            '[]'::jsonb
        ) AS models
    FROM channel_model_pricing cmp
    JOIN startwork_deepseek_target_channels target
      ON target.channel_id = cmp.channel_id
     AND target.pricing_platform = LOWER(BTRIM(cmp.platform))
    CROSS JOIN LATERAL jsonb_array_elements_text(cmp.models) AS model_name
    GROUP BY cmp.id
)
UPDATE channel_model_pricing cmp
SET models = rewritten.models,
    updated_at = NOW()
FROM rewritten
WHERE cmp.id = rewritten.id
  AND jsonb_array_length(rewritten.models) > 0
  AND cmp.models IS DISTINCT FROM rewritten.models;

INSERT INTO channel_model_pricing (
    channel_id,
    platform,
    models,
    billing_mode,
    input_price,
    output_price,
    cache_write_price,
    cache_read_price,
    time_pricing
)
SELECT
    target.channel_id,
    target.pricing_platform,
    pricing.models,
    'token',
    pricing.input_price,
    pricing.output_price,
    pricing.cache_write_price,
    pricing.cache_read_price,
    '{
      "timezone": "Asia/Shanghai",
      "periods": [
        {"start_time": "09:00", "end_time": "12:00", "multiplier": 2},
        {"start_time": "14:00", "end_time": "18:00", "multiplier": 2}
      ]
    }'::jsonb
FROM startwork_deepseek_target_channels target
CROSS JOIN (
    VALUES
        (
            '["deepseek-v4-flash","deepseek-chat","deepseek-reasoner"]'::jsonb,
            0.00000022::numeric,
            0.00000066::numeric,
            0.00000022::numeric,
            0.000000007::numeric
        ),
        (
            '["deepseek-v4-pro"]'::jsonb,
            0.00000066::numeric,
            0.00000198::numeric,
            0.00000066::numeric,
            0.000000022::numeric
        )
) AS pricing(models, input_price, output_price, cache_write_price, cache_read_price);
