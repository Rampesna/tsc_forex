//
//  Skeleton.swift
//  Facebook-style shimmer placeholders. The card/layout renders immediately and
//  its content blocks shimmer until real data arrives — a polished, professional
//  loading experience instead of a bare spinner.
//

import SwiftUI

// MARK: - Shimmer modifier

struct Shimmer: ViewModifier {
    @State private var phase: CGFloat = -1

    func body(content: Content) -> some View {
        content
            .overlay(
                GeometryReader { geo in
                    let w = geo.size.width
                    LinearGradient(
                        colors: [.clear, Color.white.opacity(0.16), .clear],
                        startPoint: .leading, endPoint: .trailing
                    )
                    .frame(width: w * 0.6)
                    .offset(x: phase * (w * 1.6))
                }
                .mask(content)
            )
            .onAppear {
                withAnimation(.linear(duration: 1.15).repeatForever(autoreverses: false)) {
                    phase = 1
                }
            }
    }
}

extension View {
    func shimmer() -> some View { modifier(Shimmer()) }

    @ViewBuilder func shimmerIf(_ active: Bool) -> some View {
        if active { modifier(Shimmer()) } else { self }
    }
}

// MARK: - Building blocks

struct SkeletonBlock: View {
    var width: CGFloat? = nil
    var height: CGFloat = 12
    var corner: CGFloat = 6

    var body: some View {
        RoundedRectangle(cornerRadius: corner, style: .continuous)
            .fill(Theme.surfaceHi)
            .frame(width: width, height: height)
    }
}

// A single market-row placeholder that mirrors MarketRow's layout.
struct MarketRowSkeleton: View {
    var body: some View {
        HStack(spacing: 10) {
            Circle().fill(Theme.surfaceHi).frame(width: 36, height: 36)
            VStack(alignment: .leading, spacing: 6) {
                SkeletonBlock(width: 120, height: 13)
                SkeletonBlock(width: 60, height: 10)
            }
            Spacer()
            SkeletonBlock(width: 44, height: 22)
            VStack(alignment: .trailing, spacing: 6) {
                SkeletonBlock(width: 70, height: 13)
                SkeletonBlock(width: 44, height: 12, corner: 5)
            }
        }
        .padding(.horizontal, 16)
        .frame(height: 62)
    }
}

// A full market list skeleton (used while the socket/snapshot loads).
struct MarketListSkeleton: View {
    var rows: Int = 10
    var body: some View {
        VStack(spacing: 0) {
            ForEach(0..<rows, id: \.self) { _ in
                MarketRowSkeleton()
                Divider().overlay(Theme.stroke).padding(.leading, 62)
            }
        }
        .shimmer()
        .padding(.top, 4)
    }
}

// Detail-screen skeleton: price header + stats + chart, all shimmering.
struct DetailSkeleton: View {
    var body: some View {
        VStack(spacing: 16) {
            VStack(alignment: .leading, spacing: 8) {
                SkeletonBlock(width: 180, height: 34, corner: 8)
                SkeletonBlock(width: 120, height: 16)
            }
            .frame(maxWidth: .infinity, alignment: .leading)

            Card(padding: 14) {
                HStack(spacing: 10) {
                    ForEach(0..<4, id: \.self) { _ in
                        VStack(alignment: .leading, spacing: 6) {
                            SkeletonBlock(width: 50, height: 10)
                            SkeletonBlock(width: 64, height: 14)
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
            }

            Card(padding: 14) {
                VStack(spacing: 12) {
                    SkeletonBlock(width: 150, height: 30, corner: 8)
                    SkeletonBlock(height: 220, corner: 10)
                }
            }
        }
        .shimmer()
    }
}
