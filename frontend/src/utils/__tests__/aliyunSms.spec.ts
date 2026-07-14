import { describe, expect, it } from "vitest";
import { isAliyunSmsRegistrationConfigured } from "../aliyunSms";

describe("isAliyunSmsRegistrationConfigured", () => {
  it("accepts verification-code service configuration", () => {
    expect(
      isAliyunSmsRegistrationConfigured({
        aliyun_sms_access_key_id: "access-key-id",
        aliyun_sms_access_key_secret_configured: true,
        aliyun_sms_registration_mode: "verify_code",
        aliyun_sms_verify_code_sign_name: "verification-sign",
        aliyun_sms_verify_code_template_code: "100001",
      }),
    ).toBe(true);
  });

  it("accepts ordinary template mode configuration", () => {
    expect(
      isAliyunSmsRegistrationConfigured({
        aliyun_sms_access_key_id: "access-key-id",
        aliyun_sms_access_key_secret: "access-key-secret",
        aliyun_sms_registration_mode: "template",
        aliyun_sms_sign_name: "template-sign",
        aliyun_sms_template_code: "SMS_123456",
      }),
    ).toBe(true);
  });

  it("rejects configuration missing credentials", () => {
    expect(
      isAliyunSmsRegistrationConfigured({
        aliyun_sms_registration_mode: "verify_code",
        aliyun_sms_verify_code_sign_name: "verification-sign",
        aliyun_sms_verify_code_template_code: "100001",
      }),
    ).toBe(false);
  });

  it("requires fields for the selected registration mode", () => {
    expect(
      isAliyunSmsRegistrationConfigured({
        aliyun_sms_access_key_id: "access-key-id",
        aliyun_sms_access_key_secret_configured: true,
        aliyun_sms_registration_mode: "template",
        aliyun_sms_verify_code_sign_name: "verification-sign",
        aliyun_sms_verify_code_template_code: "100001",
      }),
    ).toBe(false);
  });
});
