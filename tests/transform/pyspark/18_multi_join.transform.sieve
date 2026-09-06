// Multiple sections with a join chain (A + B + C)
employees
    from s3
        bucket hr
        region us-east-1
        prefix employees
        format delta

    extract
        json select message
            id id bigint
            name name string
            dept_id dept_id bigint

departments
    from s3
        bucket hr
        region us-east-1
        prefix departments
        format delta

    extract
        json select message
            id id bigint
            dept_name dept_name string

salaries
    from s3
        bucket hr
        region us-east-1
        prefix salaries
        format delta

    extract
        json select message
            emp_id emp_id bigint
            salary salary decimal(18,2)

emp_dept = employees /\ departments -> employees.dept_id = departments.id
    select
        employees.id id
        employees.name name
        departments.dept_name dept_name

emp_dept_sal = emp_dept (e) /\ salaries (s) -> e.id = s.emp_id
    select
        e.id id
        e.name name
        e.dept_name dept_name
        s.salary salary