import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "./api";
import { installEquipmentTestHost } from "./test-host";

describe("equipment maintenance host API", () => {
  const request = vi.fn();

  beforeEach(() => {
    request.mockReset();
    request.mockResolvedValue({ items: [], total: 0, offset: 0, limit: 200 });
    installEquipmentTestHost(request);
  });

  it("uses the host bridge for scoped plugin routes", async () => {
    await api.assets("pump");
    expect(request).toHaveBeenCalledWith("/v1/plugins/equipment_maintenance/api/assets?keyword=pump&limit=200");
  });

  it("passes state-changing requests through the host bridge", async () => {
    await api.createMovement({ sparePartId: "spare-1", movementType: "inbound", quantity: 3, idempotencyKey: "move-1" });
    expect(request).toHaveBeenCalledWith("/v1/plugins/equipment_maintenance/api/spare-movements", {
      method: "POST",
      body: { sparePartId: "spare-1", movementType: "inbound", quantity: 3, idempotencyKey: "move-1" }
    });
  });

  it("fails closed when the current host identity is absent", () => {
    delete window.__SKOLL_HOST__;
    expect(() => api.dashboard()).toThrow("Skoll plugin host is unavailable");
  });
});
