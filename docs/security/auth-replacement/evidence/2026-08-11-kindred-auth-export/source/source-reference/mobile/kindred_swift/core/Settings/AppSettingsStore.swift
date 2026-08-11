import Foundation
import Observation
import os
import Security
import SwiftUI
import UIKit

private enum KeychainStore {
    static let defaultService = "com.radlof.kindred-swift"
    private static let logger = Logger(subsystem: "com.radlof.kindred-swift", category: "Auth")

    static func read(_ account: String, service: String = defaultService) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]
        var result: AnyObject?
        guard SecItemCopyMatching(query as CFDictionary, &result) == errSecSuccess,
              let data = result as? Data else {
            return nil
        }
        return String(data: data, encoding: .utf8)
    }

    @discardableResult
    static func write(_ value: String, account: String, service: String = defaultService) -> OSStatus {
        let data = Data(value.utf8)
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account
        ]
        let attributes: [String: Any] = [
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
        ]
        let status = SecItemUpdate(query as CFDictionary, attributes as CFDictionary)
        if status == errSecItemNotFound {
            var item = query
            item[kSecValueData as String] = data
            item[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
            let addStatus = SecItemAdd(item as CFDictionary, nil)
            if addStatus != errSecSuccess {
                logger.error("Keychain write failed account=\(account, privacy: .public) status=\(addStatus)")
            }
            return addStatus
        }
        if status != errSecSuccess {
            logger.error("Keychain update failed account=\(account, privacy: .public) status=\(status)")
        }
        return status
    }

    @discardableResult
    static func delete(_ account: String, service: String = defaultService) -> OSStatus {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account
        ]
        return SecItemDelete(query as CFDictionary)
    }
}

/// Single source of truth for tokens. Persists to Keychain on every write,
/// notifies observers (AppSettingsStore) so SwiftUI reacts. The APIClient
/// holds this directly via the AuthTokenStore protocol; AppSettingsStore
/// exposes computed properties that read from here.
final class AuthTokenHolder: AuthTokenStore, @unchecked Sendable {
    private let lock = NSLock()
    private var token: String?
    private var refresh: String?
    private let device: String
    private let keychainService: String
    private var changeHandler: (@Sendable () -> Void)?

    init(
        token: String? = nil,
        refreshToken: String? = nil,
        deviceID: String,
        keychainService: String = "com.radlof.kindred-swift"
    ) {
        self.token = token
        self.refresh = refreshToken
        self.device = deviceID
        self.keychainService = keychainService
    }

    var deviceID: String { device }

    var authToken: String? {
        lock.lock(); defer { lock.unlock() }
        return token
    }

    var refreshToken: String? {
        lock.lock(); defer { lock.unlock() }
        return refresh
    }

    func setTokens(accessToken: String, refreshToken: String?) {
        lock.lock()
        token = accessToken
        refresh = refreshToken
        let handler = changeHandler
        lock.unlock()
        KeychainStore.write(accessToken, account: AppSettingsStore.authTokenKey, service: keychainService)
        if let refreshToken {
            KeychainStore.write(refreshToken, account: AppSettingsStore.refreshTokenKey, service: keychainService)
        } else {
            KeychainStore.delete(AppSettingsStore.refreshTokenKey, service: keychainService)
        }
        handler?()
    }

    func clearTokens() {
        lock.lock()
        token = nil
        refresh = nil
        let handler = changeHandler
        lock.unlock()
        KeychainStore.delete(AppSettingsStore.authTokenKey, service: keychainService)
        KeychainStore.delete(AppSettingsStore.refreshTokenKey, service: keychainService)
        handler?()
    }

    fileprivate func setChangeHandler(_ handler: @escaping @Sendable () -> Void) {
        lock.lock()
        changeHandler = handler
        lock.unlock()
    }

    func clearAuthToken() {
        clearTokens()
    }
}

enum AppTheme: String, CaseIterable, Identifiable {
    case system
    case light
    case dark

    var id: String { rawValue }

    var title: LocalizedStringResource {
        switch self {
        case .system:
            return "System"
        case .light:
            return "Light"
        case .dark:
            return "Dark"
        }
    }

    var colorScheme: ColorScheme? {
        switch self {
        case .system:
            return nil
        case .light:
            return .light
        case .dark:
            return .dark
        }
    }
}

enum AppLanguage: String, CaseIterable, Identifiable {
    case system = "system"
    case english = "en"
    case turkish = "tr"

    var id: String { rawValue }

    var title: LocalizedStringResource {
        switch self {
        case .system:
            return "System"
        case .english:
            return "English"
        case .turkish:
            return "Turkish"
        }
    }

    var locale: Locale {
        switch self {
        case .system:
            return .autoupdatingCurrent
        case .english:
            return Locale(identifier: "en")
        case .turkish:
            return Locale(identifier: "tr")
        }
    }
}

@MainActor
@Observable
final class AppSettingsStore {
    private static let logger = Logger(subsystem: "com.radlof.kindred-swift", category: "Auth")

    nonisolated static let authTokenKey = "app_settings.auth_token"
    nonisolated static let refreshTokenKey = "app_settings.refresh_token"
    static let currentUserIDKey = "app_settings.current_user_id"
    static let currentUserEmailKey = "app_settings.current_user_email"
    static let currentUserDisplayNameKey = "app_settings.current_user_display_name"
    static let currentUserPhoneKey = "app_settings.current_user_phone"
    static let currentUserPhoneVerifiedKey = "app_settings.current_user_phone_verified"
    static let currentUserEmailVerifiedKey = "app_settings.current_user_email_verified"
    static let currentUserProfilePhotoURLKey = "app_settings.current_user_profile_photo_url"
    static let lastReceivedPushTokenKey = "app_settings.last_received_push_token"
    static let lastUploadedPushTokenKey = "app_settings.last_uploaded_push_token"
    static let lastUploadedPushTokenUserIDKey = "app_settings.last_uploaded_push_token_user_id"

    private enum Keys {
        static let selectedTheme = "app_settings.selected_theme"
        static let selectedLanguage = "app_settings.selected_language"
        static let deviceID = "app_settings.device_id"
        static let installMarker = "app_settings.install_marker"
    }

    /// Stable per-install identifier. Sourced from
    /// `UIDevice.identifierForVendor` when available, otherwise a generated
    /// UUID persisted in UserDefaults. Survives app restarts; resets on
    /// reinstall (which is the desired security boundary).
    let deviceID: String

    var selectedTheme: AppTheme {
        didSet {
            userDefaults.set(selectedTheme.rawValue, forKey: Keys.selectedTheme)
        }
    }

    var selectedLanguage: AppLanguage {
        didSet {
            userDefaults.set(selectedLanguage.rawValue, forKey: Keys.selectedLanguage)
        }
    }

    /// Bumped on every token mutation; reads of `authToken`/`refreshToken`
    /// touch this counter so the Observation framework re-evaluates SwiftUI
    /// views when tokens change (including refreshes triggered from the
    /// APIClient).
    private var tokenVersion: Int = 0

    var authToken: String? {
        _ = tokenVersion
        return tokenHolder.authToken
    }

    var refreshToken: String? {
        _ = tokenVersion
        return tokenHolder.refreshToken
    }

    /// Persisted across launches so owner-only UI (e.g. the request queue
    /// section in DonationListingDetailView) can be gated by ownership
    /// without an extra `/me` round-trip.
    private(set) var currentUserID: String?
    private(set) var currentUserEmail: String?
    private(set) var currentUserDisplayName: String?
    private(set) var currentUserPhone: String?
    private(set) var currentUserPhoneVerified: Bool
    private(set) var currentUserEmailVerified: Bool
    private(set) var currentUserProfilePhotoURL: String?
    private(set) var lastReceivedPushToken: String?
    private(set) var lastUploadedPushToken: String?
    private(set) var lastUploadedPushTokenUserID: String?

    let tokenHolder: AuthTokenHolder

    private let userDefaults: UserDefaults

    init(userDefaults: UserDefaults = .standard, keychainService: String = "com.radlof.kindred-swift") {
        self.userDefaults = userDefaults

        if
            let storedTheme = userDefaults.string(forKey: Keys.selectedTheme),
            let theme = AppTheme(rawValue: storedTheme)
        {
            selectedTheme = theme
        } else {
            selectedTheme = .system
        }

        if
            let storedLanguage = userDefaults.string(forKey: Keys.selectedLanguage),
            let language = AppLanguage(rawValue: storedLanguage)
        {
            selectedLanguage = language
        } else {
            selectedLanguage = .system
        }

        let hasInstallMarker = userDefaults.object(forKey: Keys.installMarker) != nil
        let storedCurrentUserID = userDefaults.string(forKey: Self.currentUserIDKey)
        let storedCurrentUserEmail = userDefaults.string(forKey: Self.currentUserEmailKey)
        var storedToken = KeychainStore.read(Self.authTokenKey, service: keychainService)
        var storedRefreshToken = KeychainStore.read(Self.refreshTokenKey, service: keychainService)

        if !hasInstallMarker {
            if storedCurrentUserID == nil, storedCurrentUserEmail == nil, (storedToken != nil || storedRefreshToken != nil) {
                Self.logger.warning("Clearing auth session at launch reason=fresh_install_stale_keychain")
                KeychainStore.delete(Self.authTokenKey, service: keychainService)
                KeychainStore.delete(Self.refreshTokenKey, service: keychainService)
                storedToken = nil
                storedRefreshToken = nil
            }
            userDefaults.set(UUID().uuidString, forKey: Keys.installMarker)
        }

        Self.logger.info(
            "Auth launch state has_install_marker=\(hasInstallMarker) has_auth_token=\(storedToken != nil) has_refresh_token=\(storedRefreshToken != nil) has_current_user_id=\(storedCurrentUserID != nil)"
        )

        currentUserID = storedCurrentUserID
        currentUserEmail = storedCurrentUserEmail
        currentUserDisplayName = userDefaults.string(forKey: Self.currentUserDisplayNameKey)
        currentUserPhone = userDefaults.string(forKey: Self.currentUserPhoneKey)
        currentUserPhoneVerified = userDefaults.bool(forKey: Self.currentUserPhoneVerifiedKey)
        currentUserEmailVerified = userDefaults.bool(forKey: Self.currentUserEmailVerifiedKey)
        currentUserProfilePhotoURL = userDefaults.string(forKey: Self.currentUserProfilePhotoURLKey)
        lastReceivedPushToken = userDefaults.string(forKey: Self.lastReceivedPushTokenKey)
        lastUploadedPushToken = userDefaults.string(forKey: Self.lastUploadedPushTokenKey)
        lastUploadedPushTokenUserID = userDefaults.string(forKey: Self.lastUploadedPushTokenUserIDKey)

        let resolvedDeviceID: String = {
            if let vendorID = UIDevice.current.identifierForVendor?.uuidString {
                return vendorID
            }
            if let cached = userDefaults.string(forKey: Keys.deviceID), !cached.isEmpty {
                return cached
            }
            let generated = UUID().uuidString
            userDefaults.set(generated, forKey: Keys.deviceID)
            return generated
        }()
        deviceID = resolvedDeviceID
        tokenHolder = AuthTokenHolder(
            token: storedToken,
            refreshToken: storedRefreshToken,
            deviceID: resolvedDeviceID,
            keychainService: keychainService
        )

        // Migrate legacy plaintext token from UserDefaults if present.
        if userDefaults.string(forKey: Self.authTokenKey) != nil {
            userDefaults.removeObject(forKey: Self.authTokenKey)
        }

        tokenHolder.setChangeHandler { [weak self] in
            Task { @MainActor [weak self] in
                self?.tokenVersion &+= 1
            }
        }
    }

    func setSession(
        token: String,
        refreshToken: String?,
        userID: String?,
        email: String? = nil,
        displayName: String? = nil,
        phone: String? = nil,
        phoneVerified: Bool? = nil,
        emailVerified: Bool? = nil,
        profilePhotoURL: String? = nil
    ) {
        tokenHolder.setTokens(accessToken: token, refreshToken: refreshToken)
        currentUserID = userID
        currentUserEmail = email
        currentUserDisplayName = displayName
        currentUserPhone = phone
        currentUserPhoneVerified = phoneVerified ?? false
        currentUserEmailVerified = emailVerified ?? false
        currentUserProfilePhotoURL = profilePhotoURL
        if let userID {
            userDefaults.set(userID, forKey: Self.currentUserIDKey)
        } else {
            userDefaults.removeObject(forKey: Self.currentUserIDKey)
        }
        if let email, !email.isEmpty {
            userDefaults.set(email, forKey: Self.currentUserEmailKey)
        } else {
            userDefaults.removeObject(forKey: Self.currentUserEmailKey)
        }
        if let displayName, !displayName.isEmpty {
            userDefaults.set(displayName, forKey: Self.currentUserDisplayNameKey)
        } else {
            userDefaults.removeObject(forKey: Self.currentUserDisplayNameKey)
        }
        if let phone, !phone.isEmpty {
            userDefaults.set(phone, forKey: Self.currentUserPhoneKey)
        } else {
            userDefaults.removeObject(forKey: Self.currentUserPhoneKey)
        }
        userDefaults.set(currentUserPhoneVerified, forKey: Self.currentUserPhoneVerifiedKey)
        userDefaults.set(currentUserEmailVerified, forKey: Self.currentUserEmailVerifiedKey)
        if let profilePhotoURL, !profilePhotoURL.isEmpty {
            userDefaults.set(profilePhotoURL, forKey: Self.currentUserProfilePhotoURLKey)
        } else {
            userDefaults.removeObject(forKey: Self.currentUserProfilePhotoURLKey)
        }
        tokenVersion &+= 1
    }

    func markPhoneVerified() {
        currentUserPhoneVerified = true
        userDefaults.set(true, forKey: Self.currentUserPhoneVerifiedKey)
        tokenVersion &+= 1
    }

    func updateCurrentPhone(_ phone: String?, verified: Bool) {
        currentUserPhone = phone
        currentUserPhoneVerified = verified
        if let phone, !phone.isEmpty {
            userDefaults.set(phone, forKey: Self.currentUserPhoneKey)
        } else {
            userDefaults.removeObject(forKey: Self.currentUserPhoneKey)
        }
        userDefaults.set(verified, forKey: Self.currentUserPhoneVerifiedKey)
        tokenVersion &+= 1
    }

    func markEmailVerified(email: String? = nil) {
        guard currentUserID != nil else { return }
        if let email, let currentUserEmail,
           currentUserEmail.caseInsensitiveCompare(email) != .orderedSame {
            return
        }
        currentUserEmailVerified = true
        userDefaults.set(true, forKey: Self.currentUserEmailVerifiedKey)
        tokenVersion &+= 1
    }

    func updateCurrentProfile(displayName: String?, profilePhotoURL: String?) {
        if let displayName, !displayName.isEmpty {
            currentUserDisplayName = displayName
            userDefaults.set(displayName, forKey: Self.currentUserDisplayNameKey)
        }
        currentUserProfilePhotoURL = profilePhotoURL
        if let profilePhotoURL, !profilePhotoURL.isEmpty {
            userDefaults.set(profilePhotoURL, forKey: Self.currentUserProfilePhotoURLKey)
        } else {
            userDefaults.removeObject(forKey: Self.currentUserProfilePhotoURLKey)
        }
        tokenVersion &+= 1
    }

    func setLastUploadedPushToken(_ token: String, userID: String) {
        setLastReceivedPushToken(token)
        lastUploadedPushToken = token
        lastUploadedPushTokenUserID = userID
        userDefaults.set(token, forKey: Self.lastUploadedPushTokenKey)
        userDefaults.set(userID, forKey: Self.lastUploadedPushTokenUserIDKey)
    }

    func setLastReceivedPushToken(_ token: String) {
        lastReceivedPushToken = token
        userDefaults.set(token, forKey: Self.lastReceivedPushTokenKey)
    }

    func clearLastUploadedPushToken() {
        lastUploadedPushToken = nil
        lastUploadedPushTokenUserID = nil
        userDefaults.removeObject(forKey: Self.lastUploadedPushTokenKey)
        userDefaults.removeObject(forKey: Self.lastUploadedPushTokenUserIDKey)
    }

    func clearSession() {
        tokenHolder.clearTokens()
        currentUserID = nil
        currentUserEmail = nil
        currentUserDisplayName = nil
        currentUserPhone = nil
        currentUserPhoneVerified = false
        currentUserEmailVerified = false
        currentUserProfilePhotoURL = nil
        userDefaults.removeObject(forKey: Self.currentUserIDKey)
        userDefaults.removeObject(forKey: Self.currentUserEmailKey)
        userDefaults.removeObject(forKey: Self.currentUserDisplayNameKey)
        userDefaults.removeObject(forKey: Self.currentUserPhoneKey)
        userDefaults.removeObject(forKey: Self.currentUserPhoneVerifiedKey)
        userDefaults.removeObject(forKey: Self.currentUserEmailVerifiedKey)
        userDefaults.removeObject(forKey: Self.currentUserProfilePhotoURLKey)
        tokenVersion &+= 1
    }
}
