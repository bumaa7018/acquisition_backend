-- Хэрэглэгдээгүй бол л устгана (захирамжид оноогдсон бол FK SET NULL болгохгүйн тулд үлдээнэ).
DELETE FROM public.decision_budget b
 WHERE b.code = 'capital'
   AND NOT EXISTS (SELECT 1 FROM public.decision_draft d WHERE d.budget_id = b.id);
