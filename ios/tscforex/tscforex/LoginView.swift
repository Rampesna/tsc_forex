//
//  LoginView.swift
//  Professional dark auth screen.
//

import SwiftUI

struct LoginView: View {
    @EnvironmentObject var auth: AuthStore
    var onContinueAsGuest: (() -> Void)?
    @State private var email = ""
    @State private var password = ""
    @State private var isRegister = false

    init(onContinueAsGuest: (() -> Void)? = nil) {
        self.onContinueAsGuest = onContinueAsGuest
    }

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()
            LinearGradient(colors: [Theme.brandDeep.opacity(0.55), Theme.accent.opacity(0.10), .clear],
                           startPoint: .top, endPoint: .center)
                .ignoresSafeArea()

            VStack(spacing: 22) {
                Spacer()
                VStack(spacing: 12) {
                    Image("BrandLogo")
                        .resizable()
                        .aspectRatio(contentMode: .fit)
                        .frame(width: 92, height: 92)
                        .clipShape(RoundedRectangle(cornerRadius: 22, style: .continuous))
                        .overlay(
                            RoundedRectangle(cornerRadius: 22, style: .continuous)
                                .stroke(Color.white.opacity(0.08), lineWidth: 1)
                        )
                        .shadow(color: Theme.brandDeep.opacity(0.6), radius: 18, y: 8)
                    Text("TSC Forex").font(.system(size: 30, weight: .bold)).foregroundStyle(Theme.text)
                    Text("Canlı Altın & Döviz Piyasası")
                        .font(.system(size: 14)).foregroundStyle(Theme.textDim)
                }

                VStack(spacing: 12) {
                    field(icon: "envelope", placeholder: "E-posta") {
                        TextField("", text: $email, prompt: Text("E-posta").foregroundColor(Theme.textFaint))
                            .textInputAutocapitalization(.never)
                            .keyboardType(.emailAddress)
                            .autocorrectionDisabled()
                            .foregroundStyle(Theme.text)
                    }
                    field(icon: "lock", placeholder: "Parola") {
                        SecureField("", text: $password, prompt: Text("Parola (min 8)").foregroundColor(Theme.textFaint))
                            .foregroundStyle(Theme.text)
                    }
                }

                if let msg = auth.errorMessage {
                    Text(msg).font(.footnote).foregroundStyle(Theme.down)
                        .multilineTextAlignment(.center)
                }

                Button {
                    Task {
                        if isRegister { await auth.register(email: email, password: password) }
                        else { await auth.login(email: email, password: password) }
                    }
                } label: {
                    if auth.isBusy { ProgressView().tint(.black) }
                    else { Text(isRegister ? "Kayıt Ol" : "Giriş Yap") }
                }
                .buttonStyle(FilledButtonStyle())
                .disabled(auth.isBusy || email.isEmpty || password.count < 8)

                dividerOr

                Button {
                    Task { await auth.loginWithGoogle() }
                } label: {
                    HStack(spacing: 10) {
                        Text("G")
                            .font(.system(size: 17, weight: .heavy, design: .rounded))
                            .foregroundStyle(Color(red: 0.26, green: 0.52, blue: 0.96))
                        Text("Google ile devam et")
                            .font(.system(size: 15, weight: .semibold))
                            .foregroundStyle(.black.opacity(0.85))
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(Color.white, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                }
                .disabled(auth.isBusy)

                Button {
                    Task { await auth.loginWithApple() }
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: "apple.logo")
                            .font(.system(size: 17, weight: .semibold))
                        Text("Apple ile devam et")
                            .font(.system(size: 15, weight: .semibold))
                    }
                    .foregroundStyle(.white)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(Color.black, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                    .overlay(RoundedRectangle(cornerRadius: 12).stroke(Color.white.opacity(0.18), lineWidth: 1))
                }
                .disabled(auth.isBusy)

                Button {
                    Task { await auth.loginWithPasskey() }
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: "person.badge.key.fill")
                            .font(.system(size: 15, weight: .semibold))
                        Text("Geçiş anahtarı ile giriş")
                            .font(.system(size: 15, weight: .semibold))
                    }
                    .foregroundStyle(Theme.text)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(Theme.surface, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                    .overlay(RoundedRectangle(cornerRadius: 12).stroke(Theme.stroke, lineWidth: 1))
                }
                .disabled(auth.isBusy)

                Button {
                    isRegister.toggle(); auth.errorMessage = nil
                } label: {
                    Text(isRegister ? "Zaten hesabım var • Giriş yap" : "Hesabın yok mu? • Kayıt ol")
                        .font(.system(size: 13, weight: .medium)).foregroundStyle(Theme.accent)
                }

                if let onContinueAsGuest {
                    Button(action: onContinueAsGuest) {
                        Text("Üye olmadan piyasaları incele")
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(Theme.textDim)
                            .padding(.vertical, 6)
                    }
                    .accessibilityHint("Canlı piyasa listesine ve varlık detaylarına giriş yapmadan devam eder")
                }

                Spacer()
            }
            .padding(.horizontal, 28)
        }
    }

    private var dividerOr: some View {
        HStack(spacing: 12) {
            Rectangle().fill(Theme.stroke).frame(height: 1)
            Text("veya").font(.system(size: 12)).foregroundStyle(Theme.textFaint)
            Rectangle().fill(Theme.stroke).frame(height: 1)
        }
    }

    @ViewBuilder
    private func field<Content: View>(icon: String, placeholder: String, @ViewBuilder content: () -> Content) -> some View {
        HStack(spacing: 10) {
            Image(systemName: icon).foregroundStyle(Theme.textDim).frame(width: 20)
            content()
        }
        .padding(14)
        .background(Theme.surface, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 12).stroke(Theme.stroke, lineWidth: 1))
    }
}
