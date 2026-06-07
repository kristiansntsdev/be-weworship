UPDATE song_requests
SET lyrics = '',
    "updatedAt" = NOW()
WHERE status = 'approved'
  AND lyrics <> '';
