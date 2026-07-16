import SwiftUI

struct InstrumentCategory: Identifiable, Hashable {
    let id: String
    let title: String
    let subtitle: String
    let icon: String

    static let all: [InstrumentCategory] = [
        .init(id: "gold", title: "Altın", subtitle: "Gram, çeyrek, tam ve diğer altınlar", icon: "circle.hexagongrid.fill"),
        .init(id: "currency", title: "Döviz", subtitle: "Dolar, euro ve diğer para birimleri", icon: "banknote.fill"),
        .init(id: "crypto", title: "Kripto Para", subtitle: "Bitcoin, Ethereum ve diğer kriptolar", icon: "bitcoinsign.circle.fill"),
        .init(id: "stock", title: "Hisse Senedi", subtitle: "BIST ve global şirket hisseleri", icon: "building.2.fill"),
    ]

    static func category(for kind: String) -> InstrumentCategory {
        all.first(where: { $0.id == kind }) ?? all[0]
    }
}

struct InstrumentCategoryPickerSheet: View {
    let instruments: [InstrumentPrice]
    let selected: InstrumentCategory?
    let onSelect: (InstrumentCategory) -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var query = ""

    private var categories: [InstrumentCategory] {
        guard !query.isEmpty else { return InstrumentCategory.all }
        return InstrumentCategory.all.filter {
            $0.title.localizedCaseInsensitiveContains(query)
                || $0.subtitle.localizedCaseInsensitiveContains(query)
        }
    }

    var body: some View {
        NavigationStack {
            List(categories) { category in
                Button {
                    onSelect(category)
                    dismiss()
                } label: {
                    InstrumentCategoryRow(
                        category: category,
                        count: instruments.count(where: { $0.kind == category.id }),
                        selected: selected?.id == category.id
                    )
                }
                .buttonStyle(.plain)
                .listRowBackground(Theme.surface)
            }
            .scrollContentBackground(.hidden)
            .background(Theme.bg)
            .navigationTitle("Varlık Türü")
            .navigationBarTitleDisplayMode(.inline)
            .searchable(text: $query, prompt: "Varlık türü ara")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Vazgeç") { dismiss() }
                }
            }
        }
        .presentationBackground(Theme.bgElevated)
    }
}

struct InstrumentPickerSheet: View {
    let category: InstrumentCategory
    let instruments: [InstrumentPrice]
    let selectedCode: String
    let onSelect: (InstrumentPrice) -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var query = ""

    private var filtered: [InstrumentPrice] {
        instruments
            .filter { $0.kind == category.id }
            .filter {
                query.isEmpty
                    || $0.displayName.localizedCaseInsensitiveContains(query)
                    || $0.code.localizedCaseInsensitiveContains(query)
            }
            .sorted {
                $0.displayName.localizedStandardCompare($1.displayName) == .orderedAscending
            }
    }

    var body: some View {
        NavigationStack {
            List(filtered) { instrument in
                Button {
                    onSelect(instrument)
                    dismiss()
                } label: {
                    InstrumentChoiceRow(
                        instrument: instrument,
                        selected: selectedCode == instrument.code
                    )
                }
                .buttonStyle(.plain)
                .listRowBackground(Theme.surface)
            }
            .overlay {
                if filtered.isEmpty {
                    ContentUnavailableView(
                        "Varlık bulunamadı",
                        systemImage: "magnifyingglass",
                        description: Text("Farklı bir adla aramayı dene.")
                    )
                }
            }
            .scrollContentBackground(.hidden)
            .background(Theme.bg)
            .navigationTitle(category.title)
            .navigationBarTitleDisplayMode(.inline)
            .searchable(text: $query, prompt: "\(category.title) ara")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Vazgeç") { dismiss() }
                }
            }
        }
        .presentationBackground(Theme.bgElevated)
    }
}

struct InstrumentMultiPickerSheet: View {
    let instruments: [InstrumentPrice]
    let maxSelection: Int
    let onDone: ([String]) -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var selected: Set<String>
    @State private var category: InstrumentCategory?
    @State private var query = ""

    init(
        instruments: [InstrumentPrice],
        selectedCodes: [String],
        maxSelection: Int,
        onDone: @escaping ([String]) -> Void
    ) {
        self.instruments = instruments
        self.maxSelection = maxSelection
        self.onDone = onDone
        _selected = State(initialValue: Set(selectedCodes))
    }

    private var filteredCategories: [InstrumentCategory] {
        guard !query.isEmpty else { return InstrumentCategory.all }
        return InstrumentCategory.all.filter {
            $0.title.localizedCaseInsensitiveContains(query)
                || $0.subtitle.localizedCaseInsensitiveContains(query)
        }
    }

    private var filteredInstruments: [InstrumentPrice] {
        guard let category else { return [] }
        return instruments
            .filter { $0.kind == category.id }
            .filter {
                query.isEmpty
                    || $0.displayName.localizedCaseInsensitiveContains(query)
                    || $0.code.localizedCaseInsensitiveContains(query)
            }
            .sorted {
                $0.displayName.localizedStandardCompare($1.displayName) == .orderedAscending
            }
    }

    var body: some View {
        NavigationStack {
            Group {
                if category == nil {
                    categoryList
                } else {
                    instrumentList
                }
            }
            .scrollContentBackground(.hidden)
            .background(Theme.bg)
            .navigationTitle(category?.title ?? "İzlenecek Varlıklar")
            .navigationBarTitleDisplayMode(.inline)
            .searchable(
                text: $query,
                prompt: category == nil ? "Varlık türü ara" : "Varlık ara"
            )
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    if category != nil {
                        Button {
                            category = nil
                            query = ""
                        } label: {
                            Label("Türler", systemImage: "chevron.left")
                        }
                    } else {
                        Button("Vazgeç") { dismiss() }
                    }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Bitti (\(selected.count)/\(maxSelection))") {
                        let ordered = instruments
                            .filter { selected.contains($0.code) }
                            .map(\.code)
                        onDone(Array(ordered.prefix(maxSelection)))
                        dismiss()
                    }
                }
            }
        }
        .presentationBackground(Theme.bgElevated)
    }

    private var categoryList: some View {
        List {
            if !selected.isEmpty {
                Section("Seçilenler") {
                    ForEach(selectedInstruments) { instrument in
                        HStack {
                            Text(instrument.displayName)
                                .font(.system(size: 14, weight: .semibold))
                                .foregroundStyle(Theme.text)
                            Spacer()
                            Button {
                                selected.remove(instrument.code)
                            } label: {
                                Image(systemName: "xmark.circle.fill")
                                    .foregroundStyle(Theme.textDim)
                            }
                            .buttonStyle(.plain)
                        }
                        .listRowBackground(Theme.surface)
                    }
                }
            }
            Section("Varlık Türleri") {
                ForEach(filteredCategories) { item in
                    Button {
                        category = item
                        query = ""
                    } label: {
                        InstrumentCategoryRow(
                            category: item,
                            count: instruments.count(where: { $0.kind == item.id }),
                            selected: false
                        )
                    }
                    .buttonStyle(.plain)
                    .listRowBackground(Theme.surface)
                }
            }
        }
    }

    private var instrumentList: some View {
        List {
            Section {
                ForEach(filteredInstruments) { instrument in
                    Button {
                        toggle(instrument.code)
                    } label: {
                        InstrumentChoiceRow(
                            instrument: instrument,
                            selected: selected.contains(instrument.code)
                        )
                    }
                    .buttonStyle(.plain)
                    .listRowBackground(Theme.surface)
                }
            } footer: {
                Text("En fazla \(maxSelection) varlık seçebilirsin.")
            }
        }
    }

    private var selectedInstruments: [InstrumentPrice] {
        instruments
            .filter { selected.contains($0.code) }
            .sorted {
                $0.displayName.localizedStandardCompare($1.displayName) == .orderedAscending
            }
    }

    private func toggle(_ code: String) {
        if selected.contains(code) {
            selected.remove(code)
        } else if selected.count < maxSelection {
            selected.insert(code)
            Haptics.tap()
        }
    }
}

private struct InstrumentCategoryRow: View {
    let category: InstrumentCategory
    let count: Int
    let selected: Bool

    var body: some View {
        HStack(spacing: 13) {
            Image(systemName: category.icon)
                .font(.system(size: 18, weight: .semibold))
                .foregroundStyle(Theme.accent)
                .frame(width: 38, height: 38)
                .background(Theme.accentSoft, in: RoundedRectangle(cornerRadius: 11))
            VStack(alignment: .leading, spacing: 2) {
                Text(category.title)
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(Theme.text)
                Text(category.subtitle)
                    .font(.system(size: 11))
                    .foregroundStyle(Theme.textDim)
            }
            Spacer()
            Text("\(count)")
                .font(Theme.number(11, weight: .semibold))
                .foregroundStyle(Theme.textFaint)
            Image(systemName: selected ? "checkmark.circle.fill" : "chevron.right")
                .foregroundStyle(selected ? Theme.up : Theme.textFaint)
        }
        .padding(.vertical, 4)
    }
}

private struct InstrumentChoiceRow: View {
    let instrument: InstrumentPrice
    let selected: Bool

    var body: some View {
        HStack(spacing: 12) {
            VStack(alignment: .leading, spacing: 3) {
                Text(instrument.displayName)
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(Theme.text)
                Text(priceText)
                    .font(Theme.number(11))
                    .foregroundStyle(Theme.textDim)
            }
            Spacer()
            Image(systemName: selected ? "checkmark.circle.fill" : "circle")
                .font(.system(size: 20))
                .foregroundStyle(selected ? Theme.up : Theme.textFaint)
        }
        .padding(.vertical, 4)
    }

    private var priceText: String {
        let symbol = instrument.isUSD ? "$" : "₺"
        return "\(symbol)\(Fmt.compactPrice(instrument.price))"
    }
}
