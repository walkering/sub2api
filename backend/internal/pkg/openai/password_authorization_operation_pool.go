package openai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

func passwordAuthorizationOperationPool() map[passwordAuthorizationFlowStepKey]passwordAuthorizationFlowStepDefinition {
	return map[passwordAuthorizationFlowStepKey]passwordAuthorizationFlowStepDefinition{
		passwordAuthorizationFlowStepAuthorize: {
			key:             passwordAuthorizationFlowStepAuthorize,
			defaultNextStep: passwordAuthorizationFlowStepExtractCode,
			execute: func(ctx context.Context, flow *passwordAuthorizationFlowContext, current passwordAuthorizationFlowPage) (passwordAuthorizationFlowStepResult, error) {
				flow.stepLogf("info", "1", "打开 OAuth 授权页", "email", flow.email)
				if err := startAuthorization(ctx, flow.session, flow.authURL); err != nil {
					flow.stepLogf("error", "1", "打开 OAuth 授权页失败: "+err.Error(), "email", flow.email)
					return passwordAuthorizationFlowStepResult{}, err
				}
				flow.stepLogf("info", "1", "打开 OAuth 授权页成功，已建立 login_session 会话", "email", flow.email)
				return passwordAuthorizationFlowStepResult{Page: current}, nil
			},
		},
		passwordAuthorizationFlowStepAuthorizeEmail: {
			key: passwordAuthorizationFlowStepAuthorizeEmail,
			execute: func(ctx context.Context, flow *passwordAuthorizationFlowContext, current passwordAuthorizationFlowPage) (passwordAuthorizationFlowStepResult, error) {
				flow.stepLogf("info", "2", "提交登录邮箱", "email", flow.email)
				continueURL, pageType, err := postAuthorizeContinue(ctx, flow.session, flow.sentinel, flow.email)
				if err != nil {
					flow.stepLogf("error", "2", "提交登录邮箱失败: "+err.Error(), "email", flow.email)
					return passwordAuthorizationFlowStepResult{}, err
				}
				page := newPasswordAuthorizationFlowPage(continueURL, pageType)
				flow.workflowKind = detectPasswordAuthorizationWorkflow(page)
				flow.stepLogf("info", "2", "提交登录邮箱成功，等待识别登录/注册工作流", "email", flow.email)
				return passwordAuthorizationFlowStepResult{
					Page:     page,
					NextStep: passwordAuthorizationWorkflowAfterAuthorize(flow.workflowKind),
				}, nil
			},
		},
		passwordAuthorizationFlowStepWarmupRegisterSession: {
			key:             passwordAuthorizationFlowStepWarmupRegisterSession,
			defaultNextStep: passwordAuthorizationFlowStepRegisterUser,
			execute: func(ctx context.Context, flow *passwordAuthorizationFlowContext, current passwordAuthorizationFlowPage) (passwordAuthorizationFlowStepResult, error) {
				flow.stepLogf("info", "3", "访问 ChatGPT 首页，初始化注册会话与 Cookies", "email", flow.email)
				if err := warmupChatGPTSession(ctx, flow.session); err != nil {
					flow.stepLogf("error", "3", "初始化注册会话失败: "+err.Error(), "email", flow.email)
					return passwordAuthorizationFlowStepResult{}, err
				}
				flow.stepLogf("info", "3", "注册会话初始化成功，复用现有 Cookies 继续注册", "email", flow.email)
				return passwordAuthorizationFlowStepResult{
					Page:     current,
					NextStep: passwordAuthorizationFlowStepRegisterUser,
				}, nil
			},
		},
		passwordAuthorizationFlowStepSendEmailOTP: {
			key:             passwordAuthorizationFlowStepSendEmailOTP,
			defaultNextStep: passwordAuthorizationFlowStepPollEmailOTP,
			execute: func(ctx context.Context, flow *passwordAuthorizationFlowContext, _ passwordAuthorizationFlowPage) (passwordAuthorizationFlowStepResult, error) {
				flow.polledEmailOTPCode = ""
				flow.passwordSubmittedAt = time.Now().UTC()
				flow.stepLogf("info", "3", "在登录链路中发送邮箱验证码", "email", flow.email)
				continueURL, pageType, err := postPasswordlessSendOTP(ctx, flow.session)
				if err != nil {
					flow.stepLogf("error", "3", "发送邮箱验证码失败: "+err.Error(), "email", flow.email)
					return passwordAuthorizationFlowStepResult{}, err
				}
				page := newPasswordAuthorizationFlowPage(continueURL, pageType)
				flow.stepLogf("info", "3", "邮箱验证码发送成功，"+describeAuthPage(page.ContinueURL, page.PageType), "email", flow.email)
				return passwordAuthorizationFlowStepResult{Page: page}, nil
			},
		},
		passwordAuthorizationFlowStepRegisterUser: {
			key:             passwordAuthorizationFlowStepRegisterUser,
			defaultNextStep: passwordAuthorizationFlowStepRegisterSendEmailOTP,
			execute: func(ctx context.Context, flow *passwordAuthorizationFlowContext, _ passwordAuthorizationFlowPage) (passwordAuthorizationFlowStepResult, error) {
				flow.passwordSubmittedAt = time.Now().UTC()
				flow.stepLogf("info", "3", "提交注册请求", "email", flow.email)
				continueURL, pageType, err := postRegisterUser(ctx, flow.session, flow.sentinel, flow.email, flow.password)
				if err != nil {
					flow.stepLogf("error", "3", "提交注册请求失败: "+err.Error(), "email", flow.email)
					return passwordAuthorizationFlowStepResult{}, err
				}
				page := newPasswordAuthorizationFlowPage(continueURL, pageType)
				flow.stepLogf("info", "3", "注册请求成功，"+describeAuthPage(page.ContinueURL, page.PageType), "email", flow.email)
				return passwordAuthorizationFlowStepResult{Page: page}, nil
			},
		},
		passwordAuthorizationFlowStepRegisterSendEmailOTP: {
			key:             passwordAuthorizationFlowStepRegisterSendEmailOTP,
			defaultNextStep: passwordAuthorizationFlowStepPollEmailOTP,
			execute: func(ctx context.Context, flow *passwordAuthorizationFlowContext, _ passwordAuthorizationFlowPage) (passwordAuthorizationFlowStepResult, error) {
				flow.polledEmailOTPCode = ""
				flow.stepLogf("info", "4", "在注册链路中发送邮箱验证码", "email", flow.email)
				continueURL, pageType, err := sendRegisterEmailOTP(ctx, flow.session)
				if err != nil {
					flow.stepLogf("error", "4", "注册邮箱验证码发送失败: "+err.Error(), "email", flow.email)
					return passwordAuthorizationFlowStepResult{}, err
				}
				page := newPasswordAuthorizationFlowPage(continueURL, pageType)
				flow.stepLogf("info", "4", "注册邮箱验证码发送成功，"+describeAuthPage(page.ContinueURL, page.PageType), "email", flow.email)
				return passwordAuthorizationFlowStepResult{Page: page}, nil
			},
		},
		passwordAuthorizationFlowStepPollEmailOTP: {
			key:             passwordAuthorizationFlowStepPollEmailOTP,
			defaultNextStep: passwordAuthorizationFlowStepVerifyEmailOTP,
			execute: func(ctx context.Context, flow *passwordAuthorizationFlowContext, current passwordAuthorizationFlowPage) (passwordAuthorizationFlowStepResult, error) {
				flow.stepLogf("info", "4", "开始轮询获取邮箱验证码", "email", flow.email)
				resendFunc := func(context.Context) error { return nil }
				switch flow.workflowKind {
				case passwordAuthorizationWorkflowRegister:
					resendFunc = func(ctx context.Context) error {
						return triggerRegisterEmailOTPResend(ctx, flow.session)
					}
				default:
					resendFunc = func(ctx context.Context) error {
						return triggerLoginEmailOTPResend(ctx, flow.session)
					}
				}
				code, _, err := waitForEmailOTPCode(
					ctx,
					flow.freeMailConfig,
					flow.email,
					flow.passwordSubmittedAt,
					flow.logf,
					extendPasswordAuthorizationStepPrefix(flow.baseStepPrefix, "4"),
					resendFunc,
				)
				if err != nil {
					flow.stepLogf("error", "4", "轮询邮箱验证码失败: "+err.Error(), "email", flow.email)
					return passwordAuthorizationFlowStepResult{}, err
				}
				flow.polledEmailOTPCode = strings.TrimSpace(code)
				flow.stepLogf("info", "4", "邮箱验证码已获取，等待提交验证", "email", flow.email)
				return passwordAuthorizationFlowStepResult{Page: current}, nil
			},
		},
		passwordAuthorizationFlowStepVerifyEmailOTP: {
			key:             passwordAuthorizationFlowStepVerifyEmailOTP,
			defaultNextStep: passwordAuthorizationFlowStepFillProfile,
			execute: func(ctx context.Context, flow *passwordAuthorizationFlowContext, current passwordAuthorizationFlowPage) (passwordAuthorizationFlowStepResult, error) {
				flow.stepLogf("info", "5", "开始提交邮箱验证码", "email", flow.email)
				if strings.TrimSpace(flow.polledEmailOTPCode) == "" {
					return passwordAuthorizationFlowStepResult{}, ErrPasswordAuthorizationEmailOTP
				}
				continueURL, pageType, err := postValidateEmailOTP(ctx, flow.session, flow.sentinel, flow.polledEmailOTPCode)
				if err != nil {
					flow.stepLogf("error", "5", "提交邮箱验证码失败: "+err.Error(), "email", flow.email)
					if errors.Is(err, ErrPasswordAuthorizationEmailOTP) {
						return passwordAuthorizationFlowStepResult{}, err
					}
					return passwordAuthorizationFlowStepResult{}, fmt.Errorf("email otp validation failed: %w", err)
				}
				flow.polledEmailOTPCode = ""
				page := newPasswordAuthorizationFlowPage(continueURL, pageType)
				flow.stepLogf("info", "5", "邮箱验证码验证成功，"+describeAuthPage(page.ContinueURL, page.PageType), "email", flow.email)
				return passwordAuthorizationFlowStepResult{Page: page}, nil
			},
		},
		passwordAuthorizationFlowStepFillProfile: {
			key:             passwordAuthorizationFlowStepFillProfile,
			defaultNextStep: passwordAuthorizationFlowStepExtractCode,
			execute: func(ctx context.Context, flow *passwordAuthorizationFlowContext, current passwordAuthorizationFlowPage) (passwordAuthorizationFlowStepResult, error) {
				flow.stepLogf("info", "5", "检测到 about-you 页面，开始填写资料", "email", flow.email)
				continueURL, pageType, err := createAccountProfile(ctx, flow.session, flow.sentinel)
				if err != nil {
					flow.stepLogf("warn", "5", "补全资料失败，继续沿用当前页面: "+err.Error(), "email", flow.email)
					return passwordAuthorizationFlowStepResult{
						Page:     current,
						NextStep: passwordAuthorizationFlowStepExtractCode,
					}, nil
				}
				page := newPasswordAuthorizationFlowPage(continueURL, pageType)
				flow.stepLogf("info", "5", "填写资料成功，"+describeAuthPage(page.ContinueURL, page.PageType), "email", flow.email)
				return passwordAuthorizationFlowStepResult{Page: page}, nil
			},
		},
		passwordAuthorizationFlowStepVerifyPhoneOTP: {
			key:             passwordAuthorizationFlowStepVerifyPhoneOTP,
			defaultNextStep: passwordAuthorizationFlowStepExtractCode,
			execute: func(ctx context.Context, flow *passwordAuthorizationFlowContext, current passwordAuthorizationFlowPage) (passwordAuthorizationFlowStepResult, error) {
				if flow.phoneConfig == nil {
					flow.stepLogf("warn", "6", "登录流程命中 add_phone/add-phone，但未配置手机号接码提供商", "email", flow.email)
					return passwordAuthorizationFlowStepResult{}, ErrPasswordAuthorizationAddPhone
				}
				flow.stepLogf("info", "6", "检测到 add_phone/add-phone，开始自动完成手机号验证", "email", flow.email)
				continueURL, pageType, err := handleAddPhoneVerification(
					ctx,
					flow.session,
					flow.phoneConfig,
					current.ContinueURL,
					current.PageType,
					flow.logf,
					extendPasswordAuthorizationStepPrefix(flow.baseStepPrefix, "6"),
				)
				if err != nil {
					flow.stepLogf("error", "6", "手机号验证失败: "+err.Error(), "email", flow.email)
					if errors.Is(err, ErrPasswordAuthorizationAddPhone) {
						return passwordAuthorizationFlowStepResult{}, err
					}
					return passwordAuthorizationFlowStepResult{}, fmt.Errorf("phone verification failed: %w", err)
				}
				page := newPasswordAuthorizationFlowPage(continueURL, pageType)
				flow.stepLogf("info", "6", "手机号验证成功，"+describeAuthPage(page.ContinueURL, page.PageType), "email", flow.email)
				return passwordAuthorizationFlowStepResult{Page: page}, nil
			},
		},
		passwordAuthorizationFlowStepExtractCode: {
			key: passwordAuthorizationFlowStepExtractCode,
			execute: func(ctx context.Context, flow *passwordAuthorizationFlowContext, current passwordAuthorizationFlowPage) (passwordAuthorizationFlowStepResult, error) {
				flow.stepLogf("info", "7", "开始提取授权码", "email", flow.email)
				code, err := extractAuthorizationCode(ctx, flow.session, current.ContinueURL)
				if err != nil {
					flow.stepLogf("error", "7", "提取授权码失败: "+err.Error(), "email", flow.email)
					return passwordAuthorizationFlowStepResult{}, err
				}
				flow.stepLogf("info", "7", "OpenAI 登录成功，已获取授权码", "email", flow.email)
				return passwordAuthorizationFlowStepResult{
					Page: newPasswordAuthorizationFlowPage("code://success?code="+code, ""),
				}, nil
			},
		},
	}
}
