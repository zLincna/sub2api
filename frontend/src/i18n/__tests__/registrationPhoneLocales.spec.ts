import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('registration phone locale keys', () => {
  it('contains the Chinese registration phone copy', () => {
    expect(zh.auth.phoneLabel).toBe('手机号')
    expect(zh.auth.phonePlaceholder).toBe('请输入手机号')
    expect(zh.auth.phoneVerificationCode).toBe('手机验证码')
    expect(zh.auth.phoneVerificationCodeHint).toBe('请输入发送到您手机的6位验证码')
    expect(zh.auth.phoneCodeSentSuccess).toBe('手机验证码已发送，请注意查收。')
    expect(zh.auth.phoneRequired).toBe('请输入手机号')
  })

  it('keeps the English registration phone copy in sync', () => {
    expect(en.auth.phoneLabel).toBe('Phone Number')
    expect(en.auth.phonePlaceholder).toBe('Enter your phone number')
    expect(en.auth.phoneVerificationCode).toBe('Phone Verification Code')
    expect(en.auth.phoneVerificationCodeHint).toBe('Enter the 6-digit code sent to your phone')
    expect(en.auth.phoneCodeSentSuccess).toBe('Phone verification code sent! Please check your messages.')
    expect(en.auth.phoneRequired).toBe('Phone number is required')
  })
})
