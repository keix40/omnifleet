import AsyncStorage from "@react-native-async-storage/async-storage";
import * as Location from "expo-location";
import * as TaskManager from "expo-task-manager";
import { useCallback, useEffect, useState } from "react";
import {
  ActivityIndicator,
  Button,
  FlatList,
  SafeAreaView,
  Text,
  TextInput,
  View,
} from "react-native";
import { listJobs, login, postPosition, updateJobStatus } from "./src/api";
import { LocationQueue } from "./src/locationQueue";
import type { DispatchJob } from "./src/types";

const LOCATION_TASK = "omnifleet-driver-location";
const queue = new LocationQueue();
const VEHICLE_ID = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee";

TaskManager.defineTask(LOCATION_TASK, async ({ data, error }) => {
  if (error || !data) {
    return;
  }
  const { locations } = data as { locations: Location.LocationObject[] };
  const token = await AsyncStorage.getItem("token");
  if (!token) {
    return;
  }
  for (const loc of locations) {
    queue.enqueue({
      vehicle_id: VEHICLE_ID,
      latitude: loc.coords.latitude,
      longitude: loc.coords.longitude,
      speed_mps: loc.coords.speed ?? 0,
      heading_deg: loc.coords.heading ?? 0,
      recorded_at: loc.timestamp,
    });
  }
  const batch = queue.drainBatch(5);
  for (const p of batch) {
    try {
      await postPosition(
        token,
        p.vehicle_id,
        p.latitude,
        p.longitude,
        p.speed_mps,
        p.heading_deg,
      );
    } catch {
      queue.enqueue(p);
    }
  }
});

export default function App() {
  const [email, setEmail] = useState("driver@acme.test");
  const [password, setPassword] = useState("demo-password-change-me");
  const [tenantSlug, setTenantSlug] = useState("acme-logistics");
  const [token, setToken] = useState<string | null>(null);
  const [jobs, setJobs] = useState<DispatchJob[]>([]);
  const [loading, setLoading] = useState(false);

  const refreshJobs = useCallback(async (bearer: string) => {
    const rows = await listJobs(bearer);
    setJobs(rows);
  }, []);

  useEffect(() => {
    AsyncStorage.getItem("token").then((t) => {
      if (t) {
        setToken(t);
        refreshJobs(t).catch(() => undefined);
      }
    });
  }, [refreshJobs]);

  const onLogin = async () => {
    setLoading(true);
    try {
      const resp = await login(email, password, tenantSlug);
      setToken(resp.access_token);
      await AsyncStorage.setItem("token", resp.access_token);
      await refreshJobs(resp.access_token);
    } finally {
      setLoading(false);
    }
  };

  const startBackground = async () => {
    const { status } = await Location.requestForegroundPermissionsAsync();
    if (status !== "granted") {
      return;
    }
    await Location.requestBackgroundPermissionsAsync();
    await Location.startLocationUpdatesAsync(LOCATION_TASK, {
      accuracy: Location.Accuracy.Balanced,
      timeInterval: 5000,
      distanceInterval: 25,
      deferredUpdatesInterval: 10000,
      showsBackgroundLocationIndicator: true,
    });
  };

  if (!token) {
    return (
      <SafeAreaView style={{ flex: 1, padding: 16, gap: 8 }}>
        <Text style={{ fontSize: 22, fontWeight: "600" }}>OmniFleet Driver</Text>
        <TextInput placeholder="Tenant slug" value={tenantSlug} onChangeText={setTenantSlug} />
        <TextInput placeholder="Email" value={email} onChangeText={setEmail} autoCapitalize="none" />
        <TextInput
          placeholder="Password"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
        />
        <Button title={loading ? "Signing in…" : "Sign in"} onPress={onLogin} disabled={loading} />
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={{ flex: 1, padding: 16 }}>
      <Text style={{ fontSize: 20, fontWeight: "600" }}>Assigned jobs</Text>
      <Button title="Refresh" onPress={() => refreshJobs(token)} />
      <Button title="Start background GPS" onPress={startBackground} />
      {loading && <ActivityIndicator />}
      <FlatList
        data={jobs}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => (
          <View style={{ paddingVertical: 8, borderBottomWidth: 1, borderColor: "#ddd" }}>
            <Text>
              {item.pickup_label ?? "Pickup"} → {item.dropoff_label ?? "Dropoff"}
            </Text>
            <Text>Status: {item.status}</Text>
            <View style={{ flexDirection: "row", gap: 8, marginTop: 4 }}>
              {["en_route", "picked_up", "delivered"].map((st) => (
                <Button
                  key={st}
                  title={st}
                  onPress={() =>
                    updateJobStatus(token, item.id, st).then(() => refreshJobs(token))
                  }
                />
              ))}
            </View>
          </View>
        )}
      />
    </SafeAreaView>
  );
}
