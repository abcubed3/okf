def attest(receipt):
    return {"ok": bool(receipt.get("job_id") and receipt.get("executed_sql"))}
