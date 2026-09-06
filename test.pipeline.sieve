// Position extraction section
position
    from s3
        bucket prd_tables
        region us-east-1
        prefix position
        format delta

    extract
        json select message
            client.name name string
            client.id id bigint
            client.document document string
            asset.name asset_name string
            tradeId trade_id bigint
            positionId position_id bigint
            positionDate position_date date(YYYY-MM-DD)

// Trade extraction section
trade
    from s3
        bucket prd_tables
        region us-east-1
        prefix trde
        format delta

    extract
        json select message
            party.name party_name string
            party.id party_id bigint
            party.document party_document string
            asset.name asset_name string
            tradeId trade_id bigint

// PerAcquisition extraction section
perAcquisition
    from s3
        bucket prd_tables
        region us-east-1
        prefix trade
        format delta

    extract
        json explode message.perAcquisition array
            reference cod_fatura_origem string
            buyDate data_compra date(YYYY-MM-DD)
            quantity quantidade decimal(18,2)
            price preco decimal(18,2)
            quantity * price financeiro decimal(18,2)
            taxes taxes string

    extract
        json select message
            tradeId trade_id bigint

// Taxes extraction from perAcquisition
taxes
    from perAcquisition

    extract
        json extract perAcquisition.taxes array
            tax tax string
            amount amount decimal(18,2)
        json select perAcquisition
            reference string

// Join trade with perAcquisition
trade_perAcquisition = trade /\ perAcquisition -> trade.trade_id = perAcquisition.trade_id
    select
        trade.trade_id trade_id
        trade.party.name party_name
        trade.party.id party_id
        trade.party.document party_document
        perAcquisition.reference reference
        perAcquisition.buyDate buy_date
        perAcquisition.financeiro financeiro

// Join trade_perAcquisition with taxes
trade_perAcquisition_taxes = trade_perAcquisition (tpa) /\ taxes (tax) -> tpa.reference = taxes.reference
    select
        tpa.trade_id
        tpa.party_name
        tpa.party_id
        tpa.party_document
        tpa.reference
        tpa.buy_date
        tpa.financeiro
        tax.tax
        tax.amount

// Final transformation: position_exploded
position_exploded = position (p) /\ exploded (e) -> p.trade_id = e.trade_id, left

    transform
        financeiro = e.financeiro
        | cast decimal(18,8) decimal(18,8)
        | default(0) decimal(18,8)
        | cast string
        | replace(".", ",")
        | prefix("R$")
        | hash()

    select
        p.name string
        p.asset_name string
        p.position_id bigint
        e.trade_id bigint
        e.cod_fatura_orig_string
        e.financeiro decimal(18,8)

    to s3
        bucket prd_tables
        region us-east-1
        prefix position_exploded
        format delta
        mode overwrite
        partitioned by year, month, day