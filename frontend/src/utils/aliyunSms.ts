export interface AliyunSmsRegistrationConfig {
  aliyun_sms_access_key_id?: string;
  aliyun_sms_access_key_secret?: string;
  aliyun_sms_access_key_secret_configured?: boolean;
  aliyun_sms_registration_mode?: string;
  aliyun_sms_sign_name?: string;
  aliyun_sms_template_code?: string;
  aliyun_sms_verify_code_sign_name?: string;
  aliyun_sms_verify_code_template_code?: string;
}

export function isAliyunSmsRegistrationConfigured(
  config: AliyunSmsRegistrationConfig,
): boolean {
  const hasAccessKeyId = Boolean(config.aliyun_sms_access_key_id?.trim());
  const hasAccessKeySecret = Boolean(
    config.aliyun_sms_access_key_secret?.trim() ||
      config.aliyun_sms_access_key_secret_configured,
  );
  const registrationMode =
    config.aliyun_sms_registration_mode || "verify_code";
  const hasRegistrationTemplate =
    registrationMode === "template"
      ? Boolean(
          config.aliyun_sms_sign_name?.trim() &&
            config.aliyun_sms_template_code?.trim(),
        )
      : Boolean(
          config.aliyun_sms_verify_code_sign_name?.trim() &&
            config.aliyun_sms_verify_code_template_code?.trim(),
        );

  return hasAccessKeyId && hasAccessKeySecret && hasRegistrationTemplate;
}
