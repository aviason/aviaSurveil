import SwiftGraphQLClient
import Foundation

@MainActor
final class AuthService {
    private let graphQLClient: KindredOperationGraphQLClient?
    private let publicAuthGraphQLClient: KindredOperationGraphQLClient?
    private let settingsStore: AppSettingsStore
    private let keyRegistration: KeyRegistration?
    private let vault: PrivateKeyVault?
    private let identityBackup: IdentityBackup?
    private let analyticsClient: AnalyticsClient?
    private let pushTokenUnregisterer: PushTokenUnregistering?

    init(
        apiClient: APIClient,
        graphQLClient: KindredOperationGraphQLClient? = nil,
        publicAuthGraphQLClient: KindredOperationGraphQLClient? = nil,
        settingsStore: AppSettingsStore,
        keyRegistration: KeyRegistration? = nil,
        vault: PrivateKeyVault? = nil,
        identityBackup: IdentityBackup? = nil,
        analyticsClient: AnalyticsClient? = nil,
        pushTokenUnregisterer: PushTokenUnregistering? = nil
    ) {
        self.graphQLClient = graphQLClient
        self.publicAuthGraphQLClient = publicAuthGraphQLClient
        self.settingsStore = settingsStore
        self.keyRegistration = keyRegistration
        self.vault = vault
        self.identityBackup = identityBackup
        self.analyticsClient = analyticsClient
        self.pushTokenUnregisterer = pushTokenUnregisterer
    }

    func login(email: String, password: String) async throws {
        let input = LoginRequestDTO(
            email: email,
            password: password,
            deviceId: settingsStore.deviceID
        )
        let response: AuthResponseDTO
        response = try await loginWithSwiftGraphQL(input, graphQLClient: requirePublicAuthGraphQLClient())
        await applyAuthenticatedSession(response, password: password, registrationConsents: nil)
    }

    func reactivate(email: String, password: String) async throws {
        let input = ReactivateRequestDTO(
            email: email,
            password: password,
            deviceId: settingsStore.deviceID
        )
        let response: AuthResponseDTO
        response = try await reactivateWithSwiftGraphQL(input, graphQLClient: requirePublicAuthGraphQLClient())
        await applyAuthenticatedSession(response, password: password, registrationConsents: nil)
    }

    func register(
        email: String,
        password: String,
        displayName: String,
        phone: String,
        city: String? = nil,
        birthYear: Int? = nil,
        gender: String? = nil,
        consents: [String: Bool] = [:]
    ) async throws {
        let input = RegisterRequestDTO(
            email: email,
            password: password,
            displayName: displayName,
            phone: phone,
            deviceId: settingsStore.deviceID,
            city: city,
            birthYear: birthYear,
            gender: gender,
            consents: consents
        )
        let response: AuthResponseDTO
        response = try await registerWithSwiftGraphQL(input, graphQLClient: requirePublicAuthGraphQLClient())
        await applyAuthenticatedSession(response, password: password, registrationConsents: consents)
    }

    func startPhoneVerification() async throws -> StartPhoneVerificationResponseDTO {
        try await startPhoneVerificationWithSwiftGraphQL(graphQLClient: requireGraphQLClient())
    }

    func verifyPhone(code: String) async throws {
        let input = VerifyPhoneRequestDTO(code: code)
        try await verifyPhoneWithSwiftGraphQL(input, graphQLClient: requireGraphQLClient())
        settingsStore.markPhoneVerified()
    }

    func startPhoneChange(newPhone: String, password: String) async throws -> StartPhoneVerificationResponseDTO {
        let input = StartPhoneChangeRequestDTO(newPhone: newPhone, password: password)
        return try await startPhoneChangeWithSwiftGraphQL(input, graphQLClient: requireGraphQLClient())
    }

    func verifyPhoneChange(code: String) async throws {
        let input = VerifyPhoneChangeRequestDTO(code: code)
        let user = try await verifyPhoneChangeWithSwiftGraphQL(input, graphQLClient: requireGraphQLClient())
        settingsStore.updateCurrentPhone(user.phone, verified: user.phoneVerified ?? true)
    }

    func currentConsentValues() async -> [String: Bool] {
        guard let analyticsClient else { return [:] }
        return await analyticsClient.currentConsentValues()
    }

    func updateConsent(purpose: AnalyticsConsentPurpose, granted: Bool) async throws {
        try await analyticsClient?.updateConsents([purpose.rawValue: granted])
    }

    func verifyEmail(email: String, token: String) async throws {
        let input = VerifyEmailRequestDTO(email: email, token: token)
        try await verifyEmailWithSwiftGraphQL(input, graphQLClient: requirePublicAuthGraphQLClient())
        settingsStore.markEmailVerified(email: email)
    }

    func resendEmailVerification(email: String) async throws {
        let input = ResendEmailVerificationRequestDTO(email: email)
        try await resendEmailVerificationWithSwiftGraphQL(input, graphQLClient: requirePublicAuthGraphQLClient())
    }

    func requestPasswordReset(email: String) async throws {
        let input = ForgotPasswordRequestDTO(email: email)
        try await forgotPasswordWithSwiftGraphQL(input, graphQLClient: requirePublicAuthGraphQLClient())
    }

    func resetPassword(email: String, token: String, newPassword: String) async throws {
        let input = ResetPasswordRequestDTO(email: email, token: token, newPassword: newPassword)
        try await resetPasswordWithSwiftGraphQL(input, graphQLClient: requirePublicAuthGraphQLClient())
    }

    private func applyAuthenticatedSession(
        _ response: AuthResponseDTO,
        password: String,
        registrationConsents: [String: Bool]?
    ) async {
        settingsStore.setSession(
            token: response.token,
            refreshToken: response.refreshToken,
            userID: response.user?.id,
            email: response.user?.email,
            displayName: response.user?.displayName,
            phone: response.user?.phone,
            phoneVerified: response.user?.phoneVerified,
            emailVerified: response.user?.emailVerified
        )
        if let userID = response.user?.id {
            try? vault?.setActiveUser(userID)
            // Reconcile before registering keys: restore may seed the same
            // private key whose public half already exists server-side.
            let identityBackupOutcome: IdentityBackup.Outcome?
            do {
                identityBackupOutcome = try await identityBackup?.reconcileOnLogin(password: password)
            } catch {
                identityBackupOutcome = nil
            }
            await keyRegistration?.ensureRegistered(
                userID: userID,
                forceRatchetPrekeyUpload: identityBackupOutcome == .restoredFromBackup
            )
        }
        if let registrationConsents {
            analyticsClient?.setConsents(registrationConsents)
        } else {
            await analyticsClient?.refreshConsents()
        }
        await analyticsClient?.trackSessionStarted()
    }

    private func registerWithSwiftGraphQL(
        _ input: RegisterRequestDTO,
        graphQLClient: KindredOperationGraphQLClient
    ) async throws -> AuthResponseDTO {
        let data = try await graphQLClient.perform(
            KindredAPI.RegisterMutation(input: try input.toGraphQLInput())
        )
        return data.register.fragments.authPayloadFields.toDTO()
    }

    private func loginWithSwiftGraphQL(
        _ input: LoginRequestDTO,
        graphQLClient: KindredOperationGraphQLClient
    ) async throws -> AuthResponseDTO {
        let data = try await graphQLClient.perform(
            KindredAPI.LoginMutation(input: input.toGraphQLInput())
        )
        return data.login.fragments.authPayloadFields.toDTO()
    }

    private func reactivateWithSwiftGraphQL(
        _ input: ReactivateRequestDTO,
        graphQLClient: KindredOperationGraphQLClient
    ) async throws -> AuthResponseDTO {
        let data = try await graphQLClient.perform(
            KindredAPI.ReactivateMutation(input: input.toGraphQLInput())
        )
        return data.reactivate.fragments.authPayloadFields.toDTO()
    }

    private func verifyEmailWithSwiftGraphQL(
        _ input: VerifyEmailRequestDTO,
        graphQLClient: KindredOperationGraphQLClient
    ) async throws {
        let data = try await graphQLClient.perform(
            KindredAPI.VerifyEmailMutation(input: input.toGraphQLInput())
        )
        _ = data.verifyEmail
    }

    private func resendEmailVerificationWithSwiftGraphQL(
        _ input: ResendEmailVerificationRequestDTO,
        graphQLClient: KindredOperationGraphQLClient
    ) async throws {
        let data = try await graphQLClient.perform(
            KindredAPI.ResendEmailVerificationMutation(input: input.toGraphQLInput())
        )
        _ = data.resendEmailVerification
    }

    private func forgotPasswordWithSwiftGraphQL(
        _ input: ForgotPasswordRequestDTO,
        graphQLClient: KindredOperationGraphQLClient
    ) async throws {
        let data = try await graphQLClient.perform(
            KindredAPI.ForgotPasswordMutation(input: input.toGraphQLInput())
        )
        _ = data.forgotPassword
    }

    private func resetPasswordWithSwiftGraphQL(
        _ input: ResetPasswordRequestDTO,
        graphQLClient: KindredOperationGraphQLClient
    ) async throws {
        let data = try await graphQLClient.perform(
            KindredAPI.ResetPasswordMutation(input: input.toGraphQLInput())
        )
        _ = data.resetPassword
    }

    private func startPhoneVerificationWithSwiftGraphQL(
        graphQLClient: KindredOperationGraphQLClient
    ) async throws -> StartPhoneVerificationResponseDTO {
        let data = try await graphQLClient.perform(KindredAPI.StartPhoneVerificationMutation())
        return data.startPhoneVerification.toDTO()
    }

    private func verifyPhoneWithSwiftGraphQL(
        _ input: VerifyPhoneRequestDTO,
        graphQLClient: KindredOperationGraphQLClient
    ) async throws {
        let data = try await graphQLClient.perform(
            KindredAPI.VerifyPhoneMutation(input: input.toGraphQLInput())
        )
        _ = data.verifyPhone
    }

    private func startPhoneChangeWithSwiftGraphQL(
        _ input: StartPhoneChangeRequestDTO,
        graphQLClient: KindredOperationGraphQLClient
    ) async throws -> StartPhoneVerificationResponseDTO {
        let data = try await graphQLClient.perform(
            KindredAPI.StartPhoneChangeMutation(input: input.toGraphQLInput())
        )
        return data.startPhoneChange.toDTO()
    }

    private func verifyPhoneChangeWithSwiftGraphQL(
        _ input: VerifyPhoneChangeRequestDTO,
        graphQLClient: KindredOperationGraphQLClient
    ) async throws -> UserDTO {
        let data = try await graphQLClient.perform(
            KindredAPI.VerifyPhoneChangeMutation(input: input.toGraphQLInput())
        )
        return data.verifyPhoneChange.toDTO()
    }

    private func logoutWithSwiftGraphQL(refreshToken: String?, graphQLClient: KindredOperationGraphQLClient) async throws {
        let data = try await graphQLClient.perform(
            KindredAPI.LogoutMutation(input: KindredAPI.LogoutInput(refreshToken: graphQLNullable(refreshToken)))
        )
        _ = data.logout
    }

    /// Atomic-ish password change: re-wrap the identity backup with the
    /// new password and PUT it FIRST, then ask the server to swap the
    /// password hash. If the second step fails the user can retry
    /// (server still accepts the old password, but the backup is wrapped
    /// with the new password — they'll need to enter the new one to
    /// restore on a fresh device, which matches what they tried to set).
    func changePassword(oldPassword: String, newPassword: String) async throws {
        guard let identityBackup else {
            throw AuthError.notConfigured
        }
        try await identityBackup.rewrapWithNewPassword(newPassword)
        try await changePasswordWithSwiftGraphQL(
            oldPassword: oldPassword,
            newPassword: newPassword,
            graphQLClient: requireGraphQLClient()
        )
    }

    func signOut() async {
        let refreshToken = settingsStore.tokenHolder.refreshToken
        await pushTokenUnregisterer?.unregisterLastUploadedToken()
        if let graphQLClient {
            _ = try? await logoutWithSwiftGraphQL(refreshToken: refreshToken, graphQLClient: graphQLClient)
        }
        settingsStore.clearSession()
        // Note: we do NOT call vault.clear() here on purpose. Per the
        // recovery design, leaving the user-scoped MBK on this device
        // lets the same user recover from an email-reset password flow
        // even after signing out. A different user signing in writes
        // their own user-scoped accounts and won't see the prior MBK.
    }

    func deactivate(password: String) async throws {
        try await deactivateWithSwiftGraphQL(password: password, graphQLClient: requireGraphQLClient())
        settingsStore.clearSession()
    }

    func deleteAccount(password: String) async throws -> DeleteAccountResponseDTO {
        let response = try await deleteAccountWithSwiftGraphQL(password: password, graphQLClient: requireGraphQLClient())
        analyticsClient?.clearLocalState()
        vault?.clear()
        settingsStore.clearSession()
        return response
    }

    private func requireGraphQLClient() throws -> KindredOperationGraphQLClient {
        guard let graphQLClient else { throw AuthError.notConfigured }
        return graphQLClient
    }

    private func requirePublicAuthGraphQLClient() throws -> KindredOperationGraphQLClient {
        guard let publicAuthGraphQLClient else { throw AuthError.notConfigured }
        return publicAuthGraphQLClient
    }

    private func changePasswordWithSwiftGraphQL(
        oldPassword: String,
        newPassword: String,
        graphQLClient: KindredOperationGraphQLClient
    ) async throws {
        let data = try await graphQLClient.perform(
            KindredAPI.ChangePasswordMutation(input: KindredAPI.ChangePasswordInput(
                oldPassword: oldPassword,
                newPassword: newPassword
            ))
        )
        _ = data.changePassword
    }

    private func deactivateWithSwiftGraphQL(password: String, graphQLClient: KindredOperationGraphQLClient) async throws {
        let data = try await graphQLClient.perform(
            KindredAPI.DeactivateAccountMutation(input: KindredAPI.PasswordStepUpInput(password: password))
        )
        _ = data.deactivateAccount
    }

    private func deleteAccountWithSwiftGraphQL(
        password: String,
        graphQLClient: KindredOperationGraphQLClient
    ) async throws -> DeleteAccountResponseDTO {
        let data = try await graphQLClient.perform(
            KindredAPI.DeleteAccountMutation(input: KindredAPI.PasswordStepUpInput(password: password))
        )
        return data.deleteAccount.toDTO()
    }

    enum AuthError: Error {
        case notConfigured
    }
}

private extension RegisterRequestDTO {
    func toGraphQLInput() throws -> KindredAPI.RegisterInput {
        KindredAPI.RegisterInput(
            email: email,
            password: password,
            displayName: displayName,
            phone: .some(phone),
            city: graphQLNullable(city),
            birthYear: birthYear.map { .some(Int32($0)) } ?? .none,
            gender: graphQLNullable(gender),
            deviceId: .some(deviceId),
            consents: consents.isEmpty ? .none : .some(try awsJSON(from: consents))
        )
    }
}

private extension LoginRequestDTO {
    func toGraphQLInput() -> KindredAPI.LoginInput {
        KindredAPI.LoginInput(email: email, password: password, deviceId: .some(deviceId))
    }
}

private extension ReactivateRequestDTO {
    func toGraphQLInput() -> KindredAPI.LoginInput {
        KindredAPI.LoginInput(email: email, password: password, deviceId: .some(deviceId))
    }
}

private extension VerifyEmailRequestDTO {
    func toGraphQLInput() -> KindredAPI.VerifyEmailInput {
        KindredAPI.VerifyEmailInput(email: email, token: token)
    }
}

private extension ResendEmailVerificationRequestDTO {
    func toGraphQLInput() -> KindredAPI.EmailInput {
        KindredAPI.EmailInput(email: email)
    }
}

private extension ForgotPasswordRequestDTO {
    func toGraphQLInput() -> KindredAPI.EmailInput {
        KindredAPI.EmailInput(email: email)
    }
}

private extension ResetPasswordRequestDTO {
    func toGraphQLInput() -> KindredAPI.ResetPasswordInput {
        KindredAPI.ResetPasswordInput(email: email, token: token, newPassword: newPassword)
    }
}

private extension VerifyPhoneRequestDTO {
    func toGraphQLInput() -> KindredAPI.VerifyPhoneInput {
        KindredAPI.VerifyPhoneInput(code: code)
    }
}

private extension StartPhoneChangeRequestDTO {
    func toGraphQLInput() -> KindredAPI.StartPhoneChangeInput {
        KindredAPI.StartPhoneChangeInput(newPhone: newPhone, password: password)
    }
}

private extension VerifyPhoneChangeRequestDTO {
    func toGraphQLInput() -> KindredAPI.VerifyPhoneInput {
        KindredAPI.VerifyPhoneInput(code: code)
    }
}

extension KindredAPI.AuthPayloadFields {
    func toDTO() -> AuthResponseDTO {
        AuthResponseDTO(
            token: token,
            expiresAt: KindredGraphQLScalars.date(from: expiresAt),
            refreshToken: refreshToken,
            refreshExpiresAt: KindredGraphQLScalars.date(from: refreshExpiresAt),
            user: user.toDTO()
        )
    }
}

private extension KindredAPI.AuthPayloadFields.User {
    func toDTO() -> UserDTO {
        UserDTO(
            id: id,
            email: email,
            displayName: displayName,
            phone: phone,
            phoneVerified: phoneVerified,
            emailVerified: emailVerified,
            city: city,
            birthYear: birthYear,
            gender: gender
        )
    }
}

private extension KindredAPI.StartPhoneVerificationMutation.Data.StartPhoneVerification {
    func toDTO() -> StartPhoneVerificationResponseDTO {
        StartPhoneVerificationResponseDTO(
            expiresAt: KindredGraphQLScalars.date(from: expiresAt),
            verificationCode: verificationCode
        )
    }
}

private extension KindredAPI.StartPhoneChangeMutation.Data.StartPhoneChange {
    func toDTO() -> StartPhoneVerificationResponseDTO {
        StartPhoneVerificationResponseDTO(
            expiresAt: KindredGraphQLScalars.date(from: expiresAt),
            verificationCode: verificationCode
        )
    }
}

private extension KindredAPI.VerifyPhoneChangeMutation.Data.VerifyPhoneChange {
    func toDTO() -> UserDTO {
        UserDTO(
            id: id,
            email: email,
            displayName: displayName,
            phone: phone,
            phoneVerified: phoneVerified,
            emailVerified: emailVerified,
            city: city,
            birthYear: birthYear,
            gender: gender
        )
    }
}

private extension KindredAPI.DeleteAccountMutation.Data.DeleteAccount {
    func toDTO() -> DeleteAccountResponseDTO {
        DeleteAccountResponseDTO(
            status: status,
            scheduledPurgeAt: KindredGraphQLScalars.date(from: scheduledPurgeAt)
        )
    }
}

private func graphQLNullable(_ value: String?) -> GraphQLNullable<String> {
    value.map(GraphQLNullable.some) ?? .none
}

private func awsJSON<Value: Encodable>(from value: Value) throws -> String {
    let data = try APICoding.encoder().encode(value)
    guard let json = String(data: data, encoding: .utf8) else {
        throw KindredGraphQLClientError.invalidResponse
    }
    return json
}
