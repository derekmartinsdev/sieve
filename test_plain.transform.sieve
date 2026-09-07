client_enriched
    from s3
        bucket prd_tables
        region us-east-1
        prefix client
        format delta

    extract
        plain
            id id bigint
            source source string

    extract
        json select message
            client.name name string
            client.id client_id bigint
            client.document document string