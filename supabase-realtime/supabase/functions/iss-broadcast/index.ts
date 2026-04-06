const corsHeaders = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Headers": "authorization, x-client-info, apikey, content-type",
  "Access-Control-Allow-Methods": "POST, OPTIONS",
  "Content-Type": "application/json",
};

const realtimeTopic = "iss-position";
const realtimeEvent = "iss-update";

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
    issResponse = await fetch("http://api.open-notify.org/iss-now.json");
  } catch (error) {
    return jsonResponse(
      {
        error: "Failed to reach open-notify API",
        details: error instanceof Error ? error.message : String(error),
      },
      502,
    );
  }

  if (!issResponse.ok) {
    return jsonResponse(
      {
        error: "open-notify API returned an error",
        status: issResponse.status,
      },
      502,
    );
  }

  const issData = await issResponse.json();

  const payload = {
    source: "open-notify",
    requestedAt: new Date().toISOString(),
    timestamp: Number(issData.timestamp),
    latitude: Number(issData.iss_position?.latitude),
    longitude: Number(issData.iss_position?.longitude),
    message: issData.message,
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