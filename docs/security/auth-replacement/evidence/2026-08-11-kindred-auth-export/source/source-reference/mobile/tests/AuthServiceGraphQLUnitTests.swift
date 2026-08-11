import Foundation
import Testing
@testable import kindred_swift

@Suite("AuthService SwiftGraphQL-only configuration", .tags(.unit))
@MainActor
struct AuthServiceGraphQLUnitTests {
    @Test("Login requires public SwiftGraphQL auth client")
    func loginRequiresPublicSwiftGraphQLAuthClient() async throws {
        let api = AuthLifecycleRecordingAPIClient()
        let settingsStore = try freshAuthLifecycleSettingsStore()
        let service = AuthService(apiClient: api, settingsStore: settingsStore)

        await #expect(throws: AuthService.AuthError.self) {
            try await service.login(email: "user@example.com", password: "password-1")
        }

        #expect(api.sent.isEmpty)
        #expect(settingsStore.authToken == nil)
    }

    @Test("Verify email requires public SwiftGraphQL auth client")
    func verifyEmailRequiresPublicSwiftGraphQLAuthClient() async throws {
        let api = AuthLifecycleRecordingAPIClient()
        let settingsStore = try authenticatedAuthLifecycleSettingsStore()
        let service = AuthService(apiClient: api, settingsStore: settingsStore)

        await #expect(throws: AuthService.AuthError.self) {
            try await service.verifyEmail(email: "user@example.com", token: "verify-token")
        }

        #expect(api.sent.isEmpty)
        #expect(settingsStore.currentUserEmailVerified == false)
    }

    @Test("Phone verification requires protected SwiftGraphQL client")
    func phoneVerificationRequiresProtectedGraphQLClient() async throws {
        let api = AuthLifecycleRecordingAPIClient()
        let settingsStore = try authenticatedAuthLifecycleSettingsStore()
        let service = AuthService(apiClient: api, settingsStore: settingsStore)

        await #expect(throws: AuthService.AuthError.self) {
            _ = try await service.startPhoneVerification()
        }
        await #expect(throws: AuthService.AuthError.self) {
            try await service.verifyPhone(code: "123456")
        }

        #expect(api.sent.isEmpty)
        #expect(settingsStore.currentUserPhoneVerified == false)
    }
}

@MainActor
private func authenticatedAuthLifecycleSettingsStore() throws -> AppSettingsStore {
    let settingsStore = try freshAuthLifecycleSettingsStore()
    settingsStore.setSession(
        token: "access-token",
        refreshToken: "refresh-token",
        userID: "user-1",
        email: "user@example.com"
    )
    return settingsStore
}

@MainActor
private func freshAuthLifecycleSettingsStore() throws -> AppSettingsStore {
    let suiteName = "AuthServiceGraphQLUnitTests.\(UUID().uuidString)"
    let keychainService = "AuthServiceGraphQLUnitTests.\(UUID().uuidString)"
    let userDefaults = try #require(UserDefaults(suiteName: suiteName))
    userDefaults.removePersistentDomain(forName: suiteName)
    return AppSettingsStore(userDefaults: userDefaults, keychainService: keychainService)
}

private final class AuthLifecycleRecordingAPIClient: APIClient {
    private(set) var sent: [String] = []

    func send<Response: Decodable>(_ endpoint: Endpoint<Response>) async throws -> Response {
        sent.append(endpoint.path)
        throw APIClientError.unsuccessfulStatusCode(500)
    }
}
