# Sieve Transpiler — DSL Example
# 
# Full pipeline: position source → join with trade → transformations → output

# Source definitions
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
            tradeId trade_id bigint
            positionId position_id bigint
            positionDate position_date date(YYYY-MM-DD)
            asset.name asset_name string

# Trade source
trade
    from s3
        bucket prd_tables
        region us-east-1
        prefix trade
        format delta

    extract
        json select message
            party.name party_name string
            party.id party_id bigint
            party.document party_document string
            tradeId trade_id bigint

# Acquisition source
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
            quantity * price financeiro decimal(18,2)

# Derived source from perAcquisition
taxes
    from perAcquisition

    extract
        json extract perAcquisition.taxes array
            tax tax string
            amount amount decimal(18,2)
        json select perAcquisition
            reference string