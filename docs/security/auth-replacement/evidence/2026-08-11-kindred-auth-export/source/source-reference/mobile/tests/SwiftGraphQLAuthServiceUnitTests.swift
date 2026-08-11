import SwiftGraphQLClient
import SwiftGraphQLClient
import Foundation
import Testing
@testable import kindred_swift

@Suite("AuthService SwiftGraphQL generated mutations", .tags(.unit))
@MainActor
struct SwiftGraphQLAuthServiceUnitTests {
    @Test("Login uses generated SwiftGraphQL public auth mutation")
    func loginUsesGeneratedSwiftGraphQLPublicAuthMutation() async throws {
        let api = SwiftGraphQLAuthRecordingAPIClient()
        let graphQL = SwiftGraphQLAuthRecordingClient(responses: [loginSwiftGraphQLData()])
        let settingsStore = try freshSwiftGraphQLAuthSettingsStore()
        let service = AuthService(
            apiClient: api,
            publicAuthGraphQLClient: graphQL,
            settingsStore: settingsStore
        )

        try await service.login(email: "user@example.com", password: "password-1")

        #expect(api.sent.isEmpty)
        #expect(graphQL.operationNames == ["Login"])
        let mutation = try #require(graphQL.mutations.first as? KindredAPI.LoginMutation)
        #expect(mutation.input.email == "user@example.com")
        #expect(mutation.input.password == "password-1")
        if case .some(let deviceID) = mutation.input.deviceId {
            #expect(deviceID == settingsStore.deviceID)
        } else {
            #expect(Bool(false))
        }
        #expect(settingsStore.authToken == "access-token")
        #expect(settingsStore.refreshToken == "refresh-token")
        #expect(settingsStore.currentUserID == "user-1")
    }

    @Test("Public account recovery flows use generated SwiftGraphQL mutations")
    func publicAccountRecoveryFlowsUseGeneratedSwiftGraphQLMutations() async throws {
        let api = SwiftGraphQLAuthRecordingAPIClient()
        let graphQL = SwiftGraphQLAuthRecordingClient(responses: [
            KindredAPI.VerifyEmailMutation.Data(verifyEmail: true),
            KindredAPI.ResendEmailVerificationMutation.Data(resendEmailVerification: true),
            KindredAPI.ForgotPasswordMutation.Data(forgotPassword: true),
            KindredAPI.ResetPasswordMutation.Data(resetPassword: true)
        ])
        let settingsStore = try authenticatedSwiftGraphQLAuthSettingsStore()
        let service = AuthService(
            apiClient: api,
            publicAuthGraphQLClient: graphQL,
            settingsStore: settingsStore
        )

        try await service.verifyEmail(email: "user@example.com", token: "verify-token")
        try await service.resendEmailVerification(email: "user@example.com")
        try await service.requestPasswordReset(email: "user@example.com")
        try await service.resetPassword(email: "user@example.com", token: "reset-token", newPassword: "new-password")

        #expect(api.sent.isEmpty)
        #expect(graphQL.operationNames == [
            "VerifyEmail",
            "ResendEmailVerification",
            "ForgotPassword",
            "ResetPassword"
        ])
        let verifyEmail = try #require(graphQL.mutations[0] as? KindredAPI.VerifyEmailMutation)
        let resetPassword = try #require(graphQL.mutations[3] as? KindredAPI.ResetPasswordMutation)
        #expect(verifyEmail.input.email == "user@example.com")
        #expect(verifyEmail.input.token == "verify-token")
        #expect(resetPassword.input.email == "user@example.com")
        #expect(resetPassword.input.token == "reset-token")
        #expect(resetPassword.input.newPassword == "new-password")
        #expect(settingsStore.currentUserEmailVerified == true)
    }

    @Test("Sign out uses generated SwiftGraphQL protected auth mutation")
    func signOutUsesGeneratedSwiftGraphQLProtectedAuthMutation() async throws {
        let api = SwiftGraphQLAuthRecordingAPIClient()
        let graphQL = SwiftGraphQLAuthRecordingClient(responses: [
            KindredAPI.LogoutMutation.Data(logout: true)
        ])
        let settingsStore = try authenticatedSwiftGraphQLAuthSettingsStore()
        let pushTokenUnregisterer = RecordingPushTokenUnregisterer(settingsStore: settingsStore)
        let service = AuthService(
            apiClient: api,
            graphQLClient: graphQL,
            settingsStore: settingsStore,
            pushTokenUnregisterer: pushTokenUnregisterer
        )

        await service.signOut()

        #expect(api.sent.isEmpty)
        #expect(pushTokenUnregisterer.calls == 1)
        #expect(pushTokenUnregisterer.authTokenSeen == "access-token")
        #expect(graphQL.operationNames == ["Logout"])
        let mutation = try #require(graphQL.mutations.first as? KindredAPI.LogoutMutation)
        if case .some(let refreshToken) = mutation.input.refreshToken {
            #expect(refreshToken == "refresh-token")
        } else {
            #expect(Bool(false))
        }
        #expect(settingsStore.authToken == nil)
        #expect(settingsStore.refreshToken == nil)
    }

    @Test("Sign out does not fall back to REST after SwiftGraphQL logout failure")
    func signOutDoesNotFallBackToRESTAfterSwiftGraphQLLogoutFailure() async throws {
        let api = SwiftGraphQLAuthRecordingAPIClient()
        let graphQL = SwiftGraphQLAuthRecordingClient(responses: [])
        let settingsStore = try authenticatedSwiftGraphQLAuthSettingsStore()
        let service = AuthService(
            apiClient: api,
            graphQLClient: graphQL,
            settingsStore: settingsStore
        )

        await service.signOut()

        #expect(graphQL.operationNames == ["Logout"])
        #expect(api.sent.isEmpty)
        #expect(settingsStore.authToken == nil)
        #expect(settingsStore.refreshToken == nil)
    }
}

@MainActor
private func authenticatedSwiftGraphQLAuthSettingsStore() throws -> AppSettingsStore {
    let settingsStore = try freshSwiftGraphQLAuthSettingsStore()
    settingsStore.setSession(
        token: "access-token",
        refreshToken: "refresh-token",
        userID: "user-1",
        email: "user@example.com"
    )
    return settingsStore
}

@MainActor
private func freshSwiftGraphQLAuthSettingsStore() throws -> AppSettingsStore {
    let suiteName = "SwiftGraphQLAuthServiceUnitTests.\(UUID().uuidString)"
    let keychainService = "SwiftGraphQLAuthServiceUnitTests.\(UUID().uuidString)"
    let userDefaults = try #require(UserDefaults(suiteName: suiteName))
    userDefaults.removePersistentDomain(forName: suiteName)
    return AppSettingsStore(userDefaults: userDefaults, keychainService: keychainService)
}

private func loginSwiftGraphQLData() -> KindredAPI.LoginMutation.Data {
    KindredAPI.LoginMutation.Data(
        login: KindredAPI.LoginMutation.Data.Login(
            token: "access-token",
            expiresAt: "2026-05-07T22:00:00Z",
            refreshToken: "refresh-token",
            refreshExpiresAt: "2026-06-06T22:00:00Z",
            user: authSwiftGraphQLUser()
        )
    )
}

private func authSwiftGraphQLUser() -> KindredAPI.LoginMutation.Data.Login.User {
    KindredAPI.LoginMutation.Data.Login.User(
        id: "user-1",
        email: "user@example.com",
        displayName: "User One",
        phone: "+15555550100",
        phoneVerified: true,
        emailVerified: false,
        city: "Istanbul",
        birthYear: 1994,
        gender: "unspecified"
    )
}

@MainActor
private final class RecordingPushTokenUnregisterer: PushTokenUnregistering {
    private let settingsStore: AppSettingsStore
    private(set) var calls = 0
    private(set) var authTokenSeen: String?

    init(settingsStore: AppSettingsStore) {
        self.settingsStore = settingsStore
    }

    func unregisterLastUploadedToken() async {
        calls += 1
        authTokenSeen = settingsStore.authToken
    }
}

private final class SwiftGraphQLAuthRecordingAPIClient: APIClient {
    private(set) var sent: [String] = []

    func send<Response: Decodable>(_ endpoint: Endpoint<Response>) async throws -> Response {
        sent.append(endpoint.path)
        throw APIClientError.unsuccessfulStatusCode(500)
    }
}

private final class SwiftGraphQLAuthRecordingClient: KindredOperationGraphQLClient, @unchecked Sendable {
    private var responses: [Any]
    private(set) var operationNames: [String] = []
    private(set) var mutations: [Any] = []

    init(responses: [Any]) {
        self.responses = responses
    }

    func fetch<Query: GraphQLQuery>(
        _ query: Query,
        cachePolicy: CachePolicy.Query.SingleResponse
    ) async throws -> Query.Data where Query.ResponseFormat == SingleResponseFormat {
        throw KindredGraphQLClientError.invalidResponse
    }

    func perform<Mutation: GraphQLMutation>(
        _ mutation: Mutation
    ) async throws -> Mutation.Data where Mutation.ResponseFormat == SingleResponseFormat {
        operationNames.append(Mutation.operationName)
        mutations.append(mutation)
        guard !responses.isEmpty, let data = responses.removeFirst() as? Mutation.Data else {
            throw KindredGraphQLClientError.invalidResponse
        }
        return data
    }
}
