const corsHeaders = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Headers": "authorization, x-client-info, apikey, content-type",
  "Access-Control-Allow-Methods": "POST, OPTIONS",
  "Content-Type": "application/json",
};

const realtimeTopic = "iss-position";
const realtimeEvent = "iss-update";
const issPositionUrl = "https://api.wheretheiss.at/v1/satellites/25544";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: corsHeaders,
  });
}

Deno.serve(async (request) => {
  if (request.method === "OPTIONS") {
    return new Response("ok", { headers: corsHeaders });
  }

  if (request.method !== "POST") {
    return jsonResponse({ error: "Method not allowed" }, 405);
  }

  const supabaseUrl = Deno.env.get("SUPABASE_URL");
  const serviceRoleKey = Deno.env.get("SUPABASE_SERVICE_ROLE_KEY");

  if (!supabaseUrl || !serviceRoleKey) {
    return jsonResponse(
      { error: "Missing SUPABASE_URL or SUPABASE_SERVICE_ROLE_KEY" },
      500,
    );
  }

  let issResponse: Response;

  try {
    issResponse = await fetch(issPositionUrl);
  } catch (error) {
    return jsonResponse(
      {
        error: "Failed to reach the ISS position API",
        details: error instanceof Error ? error.message : String(error),
      },
      502,
    );
  }

  if (!issResponse.ok) {
    return jsonResponse(
      {
        error: "The ISS position API returned an error",
        status: issResponse.status,
      },
      502,
    );
  }

  const issData = await issResponse.json();

  const timestamp = Number(issData.timestamp);
  const latitude = Number(issData.latitude);
  const longitude = Number(issData.longitude);

  if (
    !Number.isFinite(timestamp) || timestamp <= 0 ||
    !Number.isFinite(latitude) || latitude < -90 || latitude > 90 ||
    !Number.isFinite(longitude) || longitude < -180 || longitude > 180
  ) {
    return jsonResponse({ error: "The ISS position API returned invalid data" }, 502);
  }

  const payload = {
    source: "wheretheiss.at",
    requestedAt: new Date().toISOString(),
    timestamp,
    latitude,
    longitude,
  };

  const broadcastResponse = await fetch(
    `${supabaseUrl}/realtime/v1/api/broadcast`,
    {
      method: "POST",
      headers: {
        apikey: serviceRoleKey,
        Authorization: `Bearer ${serviceRoleKey}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        messages: [
          {
            topic: realtimeTopic,
            event: realtimeEvent,
            private: false,
            payload,
          },
        ],
      }),
    },
  );

  if (!broadcastResponse.ok) {
    return jsonResponse(
      {
        error: "Failed to broadcast ISS update",
        status: broadcastResponse.status,
        endpoint: `${supabaseUrl}/realtime/v1/api/broadcast`,
        details: await broadcastResponse.text(),
      },
      502,
    );
  }

  return jsonResponse({ ok: true, topic: realtimeTopic, event: realtimeEvent, payload });
});
