-- Align Startwork-managed DeepSeek channels with the official V4.1-Flash
-- prices announced on 2026-09-10. Since 2026-09-14 12:00 Asia/Shanghai,
-- deepseek-v4-pro is also routed and billed as V4.1-Flash.

CREATE TEMP TABLE startwork_deepseek_v41_target_channels ON COMMIT DROP AS
SELECT DISTINCT
    cg.channel_id,
    CASE
        WHEN LOWER(BTRIM(g.platform)) IN ('', 'openai_chat') THEN 'openai'
        ELSE LOWER(BTRIM(g.platform))
    END AS pricing_platform
FROM groups g
JOIN channel_groups cg ON cg.group_id = g.id
WHERE g.deleted_at IS NULL
  AND (
      g.name LIKE 'deepseek-ecosystem-%'
      OR g.name LIKE 'deepseek-private-%'
      OR g.name LIKE 'deepseek-anthropic-ecosystem-%'
      OR g.name LIKE 'deepseek-anthropic-private-%'
  );

WITH rewritten AS (
    SELECT
        cmp.id,
        COALESCE(
            jsonb_agg(to_jsonb(model_name))
                FILTER (WHERE LOWER(BTRIM(model_name)) NOT IN (
                    'deepseek-flash',
                    'deepseek-v4-flash',
                    'deepseek-v4-flash-vision-exp',
                    'deepseek-v4-pro',
                    'deepseek-chat',
                    'deepseek-reasoner'
                )),
            '[]'::jsonb
        ) AS models
    FROM channel_model_pricing cmp
    JOIN startwork_deepseek_v41_target_channels target
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
                    'deepseek-flash',
                    'deepseek-v4-flash',
                    'deepseek-v4-flash-vision-exp',
                    'deepseek-v4-pro',
                    'deepseek-chat',
                    'deepseek-reasoner'
                )),
            '[]'::jsonb
        ) AS models
    FROM channel_model_pricing cmp
    JOIN startwork_deepseek_v41_target_channels target
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
    '["deepseek-flash","deepseek-v4-flash","deepseek-v4-flash-vision-exp","deepseek-v4-pro","deepseek-chat","deepseek-reasoner"]'::jsonb,
    'token',
    0.00000015::numeric,
    0.00000060::numeric,
    0.00000015::numeric,
    0.000000003::numeric,
    '{
      "timezone": "Asia/Shanghai",
      "weekdays_only": true,
      "periods": [
        {"start_time": "09:00", "end_time": "12:00", "multiplier": 2},
        {"start_time": "14:00", "end_time": "18:00", "multiplier": 2}
      ]
    }'::jsonb
FROM startwork_deepseek_v41_target_channels target;
