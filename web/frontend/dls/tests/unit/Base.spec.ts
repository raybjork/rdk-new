import { enableAutoDestroy, mount } from "@vue/test-utils";
import Base from "@/components/Base.vue";
import ViamSelect from "@/components/ViamSelect.vue";
import ViamInput from "@/components/ViamInput.vue";
import Collapse from "@/components/Collapse.vue";
import Breadcrumbs from "@/components/Breadcrumbs.vue";
import ViamIcon from "@/components/ViamIcon.vue";
import Tabs from "@/components/Tabs.vue";
import Tab from "@/components/Tab.vue";
import RadioButtons from "@/components/RadioButtons.vue";
import ViamButton from "@/components/Button.vue";
import KeyboardInput from "@/components/KeyboardInput.vue";
import Range from "@/components/Range.vue";

describe("Base", () => {
  enableAutoDestroy(afterEach);

  it("has html structure", async () => {
    const wrapper = mount(Base, {
      propsData: {
        streamName: "Test",
        baseName: "Test",
        crumbs: ["Keyboard", "Discrete"],
      },
      components: {
        Collapse,
        Breadcrumbs,
        ViamIcon,
        RadioButtons,
        Tabs,
        Tab,
        ViamButton,
        KeyboardInput,
        ViamSelect,
        ViamInput,
        Range,
      },
    });

    expect(wrapper.element.tagName).toBe("DIV");
  });

  it("click tabs", async () => {
    const wrapper = mount(Base, {
      propsData: {
        streamName: "Test",
        baseName: "Test",
        crumbs: ["Keyboard", "Discrete"],
      },
      components: {
        Collapse,
        Breadcrumbs,
        ViamIcon,
        RadioButtons,
        Tabs,
        Tab,
        ViamButton,
        KeyboardInput,
        ViamSelect,
        ViamInput,
        Range,
      },
    });
    const firstButton = wrapper.find(".tabs-container button:first-child");
    const lastButton = wrapper.find(".tabs-container button:last-child");
    const firstOption = wrapper.find(".tabs-container button > div");
    const lastOption = wrapper.find(".tabs-container button:last-child > div");
    expect(firstOption.element.innerHTML.trim()).toBe("Keyboard");
    const keyboardColumnWrapper = wrapper.find(
      ".grid button:first-child span:last-child span"
    );
    expect(keyboardColumnWrapper.element.innerHTML.trim()).toBe("W");
    await lastButton.trigger("click");
    expect(lastOption.element.innerHTML.trim()).toBe("Discrete");
    expect(lastButton.attributes("aria-selected")).toBe("true");
    expect(firstButton.attributes("aria-selected")).toBe("false");
    const divColumnWrapper = wrapper.find(
      ".column button:first-child span:last-child"
    );
    expect(divColumnWrapper.element.innerHTML.trim()).toBe("Straight");
  });

  it("click discrete straight", async () => {
    const wrapper = mount(Base, {
      propsData: {
        streamName: "Test",
        baseName: "Test",
        crumbs: ["Keyboard", "Discrete"],
      },
      components: {
        Collapse,
        Breadcrumbs,
        ViamIcon,
        RadioButtons,
        Tabs,
        Tab,
        ViamButton,
        KeyboardInput,
        ViamSelect,
        ViamInput,
        Range,
      },
    });
    const lastButton = wrapper.find(".tabs-container button:last-child");
    await lastButton.trigger("click");
    expect(lastButton.attributes("aria-selected")).toBe("true");
    const divColumnWrapper = wrapper.find(
      ".column button:first-child span:last-child"
    );
    expect(divColumnWrapper.element.innerHTML.trim()).toBe("Straight");
    expect(wrapper.vm.$data.movementMode).toBe("Straight");
    expect(wrapper.vm.$data.movementType).toBe("Continous");
    expect(wrapper.vm.$data.direction).toBe("Forwards");
    expect(wrapper.vm.$data.spinType).toBe("Clockwise");
  });

  it("click discrete arc", async () => {
    const wrapper = mount(Base, {
      propsData: {
        streamName: "Test",
        baseName: "Test",
        crumbs: ["Keyboard", "Discrete"],
      },
      components: {
        Collapse,
        Breadcrumbs,
        ViamIcon,
        RadioButtons,
        Tabs,
        Tab,
        ViamButton,
        KeyboardInput,
        ViamSelect,
        ViamInput,
        Range,
      },
    });
    const lastButton = wrapper.find(".tabs-container button:last-child");
    await lastButton.trigger("click");
    expect(lastButton.attributes("aria-selected")).toBe("true");
    const divColumnWrapper = wrapper.find(
      ".column button:nth-child(2) span:first-child"
    );
    expect(divColumnWrapper.element.innerHTML.trim()).toBe("Arc");
    await divColumnWrapper.trigger("click");
    expect(wrapper.vm.$data.movementMode).toBe("Arc");
    expect(wrapper.vm.$data.movementType).toBe("Continous");
    expect(wrapper.vm.$data.direction).toBe("Forwards");
    expect(wrapper.vm.$data.spinType).toBe("Clockwise");
  });

  it("click discrete arc", async () => {
    const wrapper = mount(Base, {
      propsData: {
        streamName: "Test",
        baseName: "Test",
        crumbs: ["Keyboard", "Discrete"],
      },
      components: {
        Collapse,
        Breadcrumbs,
        ViamIcon,
        RadioButtons,
        Tabs,
        Tab,
        ViamButton,
        KeyboardInput,
        ViamSelect,
        ViamInput,
        Range,
      },
    });
    const lastButton = wrapper.find(".tabs-container button:last-child");
    await lastButton.trigger("click");
    expect(lastButton.attributes("aria-selected")).toBe("true");
    const divColumnWrapper = wrapper.find(
      ".column button:nth-child(2) span:first-child"
    );
    expect(divColumnWrapper.element.innerHTML.trim()).toBe("Arc");
    await divColumnWrapper.trigger("click");
    expect(wrapper.vm.$data.movementMode).toBe("Arc");
    expect(wrapper.vm.$data.movementType).toBe("Continous");
    expect(wrapper.vm.$data.direction).toBe("Forwards");
    expect(wrapper.vm.$data.spinType).toBe("Clockwise");
    const backwardColumnWrapper = wrapper.find(
      ".flex:nth-child(2) button:nth-child(2) span:first-child"
    );
    expect(backwardColumnWrapper.element.innerHTML.trim()).toBe("Backwards");
    await backwardColumnWrapper.trigger("click");
    expect(wrapper.vm.$data.direction).toBe("Backwards");
  });

  it("click discrete arc", async () => {
    const wrapper = mount(Base, {
      propsData: {
        streamName: "Test",
        baseName: "Test",
        crumbs: ["Keyboard", "Discrete"],
      },
      components: {
        Collapse,
        Breadcrumbs,
        ViamIcon,
        RadioButtons,
        Tabs,
        Tab,
        ViamButton,
        KeyboardInput,
        ViamSelect,
        ViamInput,
        Range,
      },
    });
    const lastButton = wrapper.find(".tabs-container button:last-child");
    await lastButton.trigger("click");
    expect(lastButton.attributes("aria-selected")).toBe("true");
    const divColumnWrapper = wrapper.find(
      ".column button:nth-child(2) span:first-child"
    );
    expect(divColumnWrapper.element.innerHTML.trim()).toBe("Arc");
    await divColumnWrapper.trigger("click");
    expect(wrapper.vm.$data.movementMode).toBe("Arc");
    expect(wrapper.vm.$data.movementType).toBe("Continous");
    expect(wrapper.vm.$data.direction).toBe("Forwards");
    expect(wrapper.vm.$data.spinType).toBe("Clockwise");
    const backwardColumnWrapper = wrapper.find(
      ".flex:nth-child(2) button:nth-child(2) span:first-child"
    );
    expect(backwardColumnWrapper.element.innerHTML.trim()).toBe("Backwards");
    await backwardColumnWrapper.trigger("click");
    expect(wrapper.vm.$data.direction).toBe("Backwards");
    const ccwColumnWrapper = wrapper.find(
      ".flex:nth-child(3) button:nth-child(2) span:first-child"
    );
    expect(ccwColumnWrapper.element.innerHTML.trim()).toBe("Counterclockwise");
    await ccwColumnWrapper.trigger("click");
    expect(wrapper.vm.$data.spinType).toBe("Counterclockwise");
  });

  it("click discrete spin", async () => {
    const wrapper = mount(Base, {
      propsData: {
        streamName: "Test",
        baseName: "Test",
        crumbs: ["Keyboard", "Discrete"],
      },
      components: {
        Collapse,
        Breadcrumbs,
        ViamIcon,
        RadioButtons,
        Tabs,
        Tab,
        ViamButton,
        KeyboardInput,
        ViamSelect,
        ViamInput,
        Range,
      },
    });
    const lastButton = wrapper.find(".tabs-container button:last-child");
    await lastButton.trigger("click");
    expect(lastButton.attributes("aria-selected")).toBe("true");
    const divColumnWrapper = wrapper.find(
      ".column button:nth-child(3) span:first-child"
    );
    expect(divColumnWrapper.element.innerHTML.trim()).toBe("Spin");
    await divColumnWrapper.trigger("click");
    expect(wrapper.vm.$data.movementMode).toBe("Spin");
    expect(wrapper.vm.$data.movementType).toBe("Continous");
    expect(wrapper.vm.$data.direction).toBe("Forwards");
    expect(wrapper.vm.$data.spinType).toBe("Clockwise");
  });

  it("click discrete spin counterclockwise", async () => {
    const wrapper = mount(Base, {
      propsData: {
        streamName: "Test",
        baseName: "Test",
        crumbs: ["Keyboard", "Discrete"],
      },
      components: {
        Collapse,
        Breadcrumbs,
        ViamIcon,
        RadioButtons,
        Tabs,
        Tab,
        ViamButton,
        KeyboardInput,
        ViamSelect,
        ViamInput,
        Range,
      },
    });
    const lastButton = wrapper.find(".tabs-container button:last-child");
    await lastButton.trigger("click");
    expect(lastButton.attributes("aria-selected")).toBe("true");
    const divColumnWrapper = wrapper.find(
      ".column button:nth-child(3) span:first-child"
    );
    expect(divColumnWrapper.element.innerHTML.trim()).toBe("Spin");
    await divColumnWrapper.trigger("click");
    expect(wrapper.vm.$data.movementMode).toBe("Spin");
    expect(wrapper.vm.$data.movementType).toBe("Continous");
    expect(wrapper.vm.$data.direction).toBe("Forwards");
    expect(wrapper.vm.$data.spinType).toBe("Clockwise");
    const ccwColumnWrapper = wrapper.find(
      ".flex:nth-child(3) button:nth-child(2) span:first-child"
    );
    expect(ccwColumnWrapper.element.innerHTML.trim()).toBe("Counterclockwise");
    await ccwColumnWrapper.trigger("click");
    expect(wrapper.vm.$data.spinType).toBe("Counterclockwise");
  });
});
