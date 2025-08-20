CREATE TABLE IF NOT EXISTS orders (
  -- Храним uid отдельной колнкой, а остальные данные в JSONB
  order_uid TEXT PRIMARY KEY, -- Postgres создает инедкс под primary поле
  data  JSONB NOT NULL,

  -- Поля для администрирования(не относятся к бизнес-логике)
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP

  -- Проверки на валидность данных
  CONSTRAINT data_is_object CHECK (jsonb_typeof(data) = 'object'),
  CONSTRAINT uid_matches CHECK (order_uid = data->>'order_uid')
);

-- Создаем триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at := NOW();
  RETURN NEW;
END; $$ LANGUAGE plpgsql;