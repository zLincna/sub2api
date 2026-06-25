-- 统一阿里云短信配置，并增加拼车短信通知开关与模板配置。
INSERT INTO settings (key, value, updated_at)
VALUES
  ('aliyun_sms_sign_name', '兴惠云科技', NOW()),
  ('aliyun_sms_template_code', 'SMS_506825188', NOW()),
  ('aliyun_sms_template_param_key', 'code', NOW()),
  ('aliyun_sms_template_static_params', '{}', NOW()),
  ('aliyun_sms_valid_time_seconds', '300', NOW()),
  ('aliyun_sms_interval_seconds', '60', NOW()),
  ('carpool_admin_full_sms_notify_enabled', 'false', NOW()),
  ('carpool_admin_full_sms_phones', '', NOW()),
  ('carpool_admin_full_sms_template_code', 'SMS_508550246', NOW()),
  ('carpool_user_active_sms_notify_enabled', 'false', NOW()),
  ('carpool_user_active_sms_template_code', 'SMS_508655235', NOW())
ON CONFLICT (key) DO NOTHING;
