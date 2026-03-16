import { describe, it, expect } from 'vitest'
import { loginSchema, registerSchema } from '../validation'

describe('Form Validation Schemas', () => {
  describe('loginSchema', () => {
    it('should validate correct login data', () => {
      const result = loginSchema.safeParse({
        email: 'test@example.com',
        password: 'password123',
      })

      expect(result.success).toBe(true)
    })

    it('should reject invalid email format', () => {
      const result = loginSchema.safeParse({
        email: 'invalid-email',
        password: 'password123',
      })

      expect(result.success).toBe(false)
      if (!result.success) {
        expect(result.error.errors[0].message).toContain('email')
      }
    })

    it('should reject empty email', () => {
      const result = loginSchema.safeParse({
        email: '',
        password: 'password123',
      })

      expect(result.success).toBe(false)
    })

    it('should reject empty password', () => {
      const result = loginSchema.safeParse({
        email: 'test@example.com',
        password: '',
      })

      expect(result.success).toBe(false)
    })

    it('should reject short password', () => {
      const result = loginSchema.safeParse({
        email: 'test@example.com',
        password: 'short',
      })

      expect(result.success).toBe(false)
      if (!result.success) {
        const passwordError = result.error.errors.find((e) => e.path[0] === 'password')
        expect(passwordError?.message).toContain('at least')
      }
    })

    it('should reject missing fields', () => {
      const result = loginSchema.safeParse({})

      expect(result.success).toBe(false)
    })

    it('should accept email with subdomain', () => {
      const result = loginSchema.safeParse({
        email: 'user@mail.example.com',
        password: 'password123',
      })

      expect(result.success).toBe(true)
    })

    it('should accept email with numbers', () => {
      const result = loginSchema.safeParse({
        email: 'user123@example.com',
        password: 'password123',
      })

      expect(result.success).toBe(true)
    })
  })

  describe('registerSchema', () => {
    it('should validate correct register data', () => {
      const result = registerSchema.safeParse({
        email: 'test@example.com',
        password: 'password123',
        confirmPassword: 'password123',
      })

      expect(result.success).toBe(true)
    })

    it('should validate register data with nickname', () => {
      const result = registerSchema.safeParse({
        email: 'test@example.com',
        password: 'password123',
        confirmPassword: 'password123',
        nickname: 'TestUser',
      })

      expect(result.success).toBe(true)
    })

    it('should reject invalid email format', () => {
      const result = registerSchema.safeParse({
        email: 'invalid-email',
        password: 'password123',
        confirmPassword: 'password123',
      })

      expect(result.success).toBe(false)
      if (!result.success) {
        expect(result.error.errors[0].message).toContain('email')
      }
    })

    it('should reject short password', () => {
      const result = registerSchema.safeParse({
        email: 'test@example.com',
        password: 'short',
        confirmPassword: 'short',
      })

      expect(result.success).toBe(false)
      if (!result.success) {
        const passwordError = result.error.errors.find((e) => e.path[0] === 'password')
        expect(passwordError?.message).toContain('at least')
      }
    })

    it('should reject mismatched passwords', () => {
      const result = registerSchema.safeParse({
        email: 'test@example.com',
        password: 'password123',
        confirmPassword: 'different123',
      })

      expect(result.success).toBe(false)
      if (!result.success) {
        const confirmError = result.error.errors.find((e) => e.path[0] === 'confirmPassword')
        expect(confirmError?.message).toContain('match')
      }
    })

    it('should reject empty email', () => {
      const result = registerSchema.safeParse({
        email: '',
        password: 'password123',
        confirmPassword: 'password123',
      })

      expect(result.success).toBe(false)
    })

    it('should reject empty password', () => {
      const result = registerSchema.safeParse({
        email: 'test@example.com',
        password: '',
        confirmPassword: '',
      })

      expect(result.success).toBe(false)
    })

    it('should reject missing confirmPassword', () => {
      const result = registerSchema.safeParse({
        email: 'test@example.com',
        password: 'password123',
      })

      expect(result.success).toBe(false)
    })

    it('should reject nickname that is too long', () => {
      const result = registerSchema.safeParse({
        email: 'test@example.com',
        password: 'password123',
        confirmPassword: 'password123',
        nickname: 'a'.repeat(51), // Max 50 characters
      })

      expect(result.success).toBe(false)
      if (!result.success) {
        const nicknameError = result.error.errors.find((e) => e.path[0] === 'nickname')
        expect(nicknameError?.message).toContain('at most')
      }
    })

    it('should accept nickname with valid length', () => {
      const result = registerSchema.safeParse({
        email: 'test@example.com',
        password: 'password123',
        confirmPassword: 'password123',
        nickname: 'Valid Nickname',
      })

      expect(result.success).toBe(true)
    })

    it('should handle optional nickname', () => {
      const result = registerSchema.safeParse({
        email: 'test@example.com',
        password: 'password123',
        confirmPassword: 'password123',
        nickname: undefined,
      })

      expect(result.success).toBe(true)
    })
  })
})
