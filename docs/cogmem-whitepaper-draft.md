**CogMem: A Modular Cognitive Architecture for LLM Agents with Tiered Memory, Dynamic Processing, Reflective Adaptation, Multi-Tenancy and  Scriptability**

**Abstract:** Large Language Models (LLMs) require advanced cognitive capabilities, particularly robust memory systems, to function effectively as autonomous agents capable of complex tasks, personalization, collaboration, and continuous learning. Existing memory solutions often lack multi-user/multi-agent context management and flexible customization. We propose CogMem, a comprehensive cognitive architecture synthesizing key concepts from recent research while introducing novel features for real-world deployment. CogMem integrates: 1) A modular design inspired by cognitive architectures (CogArch, SR-CIS). 2) A tiered memory system (MemGPT-style WM/LTM) managing context. 3) A sophisticated Long-Term Memory (LTM) supporting diverse content and backends (including vectors, structured data, and temporal knowledge graphs like Zep's Graphiti), architected for **multi-entity awareness** (scoping memory per user or group) and **multi-agent interaction** (supporting shared memory segments within an entity). 4) Dynamic memory processes including iterative retrieval, context-sensitive recall, and consolidation. 5) A reflective loop (SR-CIS) enabling self-correction and adaptation for self-evolution. 6) **Embedded Lua scripting (`gopher-lua`)** enabling rapid prototyping and customization of core logic (e.g., MMU strategies, reflection rules) without recompiling the core Go components. CogMem aims to provide a robust, flexible, and performant **library (primarily in Go, with possible future Python support)** for building LLM agents that exhibit enhanced coherence, adaptability, collaboration, and reasoning, positioning memory as a cornerstone for more capable AI, while remaining cognizant of the challenges in achieving true AGI and avoiding anthropomorphic overclaims (Cognitive Mirage).

**1. Introduction**

Large Language Models (LLMs) have demonstrated remarkable proficiency in natural language understanding, generation, and various reasoning tasks [Brown et al., 2020]. However, their inherent limitations – particularly the fixed context window size and stateless nature between independent interactions – significantly hinder their application in domains requiring long-term coherence, persistent memory, adaptation, collaboration, and complex multi-step reasoning [Mialon et al., 2023]. These limitations are particularly acute for LLM-based autonomous agents, which need to maintain state, learn from interactions, recall past information accurately, and adapt their behavior over extended periods [Wang et al., 2023; Xi et al., 2023]. Furthermore, real-world applications often involve multiple users or agents interacting within specific contexts (e.g., individuals within a company, multiple support bots serving a single customer) demanding sophisticated memory partitioning and sharing mechanisms.

Initial approaches like basic prompt engineering or simple Retrieval-Augmented Generation (RAG) [Lewis et al., 2020] fall short. They typically lack sophisticated memory management, struggle with coherent integration, often deal only with static documents, and do not inherently support dynamic processes, multi-tenancy, or collaborative memory access. Enterprise applications, in particular, demand dynamic knowledge integration from diverse, evolving sources, respecting organizational or user boundaries (`2501.13956`).


Recognizing these gaps, recent research has explored more structured solutions. Cognitive architectures inspired by human cognition propose modular systems with distinct components like working memory (WM) and long-term memory (LTM) [e.g., `2310.08560`; `2408.09176`]. Systems like MemGPT (`2310.08998`) introduce the concept of tiered memory management, where the LLM acts like an operating system managing its context window (akin to RAM) and interacting with external storage (akin to disk). MemoryBank (`2309.02427`) focuses on practical LTM implementations using summarization and interaction history. **Zep (`2501.13956`) introduces a memory layer centered around a temporal knowledge graph (Graphiti) designed to dynamically synthesize unstructured conversational data and structured business data while maintaining historical relationships, reporting strong performance on benchmarks reflecting enterprise needs.** Furthermore, work has emerged on dynamic memory processes like human-like recall and consolidation (`2404.00573`), iterative retrieval mechanisms (ILM-TR, `2410.12859`), and the crucial role of reflection in enabling self-correction and adaptation (SR-CIS, `2408.01970`). Concurrently, researchers emphasize memory's foundational role in achieving personalization (Second Me, `2503.08102`), lifelong learning and self-evolution (`2410.15665`), learning from experience in embodied agents (MINDSTORES, `2501.19318`), and ultimately, as a pillar for Artificial General Intelligence (AGI) (`2501.03151`). Taxonomies are also being developed to categorize the diverse memory mechanisms being explored (`2504.02441`). Comprehensive surveys map this rapidly evolving landscape [e.g., `2404.13501`, `2404.10890`].

While these advancements are significant, they often focus on specific aspects of the memory or cognition challenge. There is a need for a unified architecture that synthesizes these complementary ideas, explicitly addressing multi-entity contexts, collaborative agent memory, and enabling rapid customization for diverse application into a coherent and practical framework.

In this paper, we propose **CogMem**, a modular cognitive architecture designed to address these challenges. CogMem integrates insights from cognitive science and recent AI research to create a system featuring:
*   A **modular design** separating perception, memory, reasoning, reflection, and action.
*   A **tiered memory system** with distinct WM (LLM context) and LTM (external, persistent store).
*   An **explicit Memory Management Unit (MMU)** orchestrating information flow and context management, inspired by MemGPT.
*   A **sophisticated LTM** capable of storing diverse information (facts, dialogues, experiences/tuples, temporal relationships) using hybrid storage backends, potentially including temporal knowledge graphs.
*   **Dynamic memory processes**, including context-aware, potentially iterative retrieval and consolidation mechanisms informed by experience and reflection.
*   A **reflective loop** enabling the agent to analyze its performance and adapt its knowledge and strategies over time, fostering self-improvement.
*   **Embedded Lua scripting ( `gopher-lua` )** allowing key logical components (e.g., MMU retrieval filters, reflection triggers) to be defined in scripts for fast prototyping and flexible customization.

Our primary contribution is the proposal of this synthesized architecture, outlining its components, interactions, and potential benefits. We aim for CogMem to serve as a blueprint and eventually an open-source library, providing a flexible foundation for building more capable, adaptive, and coherent LLM agents. We position CogMem as a functional enhancement, carefully distinguishing its capabilities from unsubstantiated claims of human-like cognition, acknowledging the important critique of the "Cognitive Mirage" (`2311.02982`).


**2. Related Work**

CogMem synthesizes and extends several research threads:

*   **Cognitive Architectures for LLMs:** We adopt the modular philosophy from works like `2310.08560`, `2408.09176`, and the decoupling/reflection concepts from SR-CIS (`2408.01970`). CogMem extends these by embedding multi-entity context management directly into the architectural fabric.

*   **Memory System Implementations & Management:** We incorporate the tiered WM/LTM structure and MMU concept from MemGPT (`2310.08998`). Our LTM design supports diverse backends, drawing inspiration from practical systems like MemoryBank (`2309.02427`) and advanced graph-based approaches like Zep (`2501.13956`) with its temporal awareness. CogMem distinguishes itself by structuring the LTM explicitly for multi-entity partitioning and controlled sharing between agents, a layer often implicit or absent in other systems. The taxonomy in `2504.02441` helps position CogMem's primary focus on architectural/external memory, while acknowledging other types.

*   **Dynamic Memory Processes:** We incorporate ideas for dynamic recall/consolidation (`2404.00573`), iterative retrieval (ILM-TR, `2410.12859`), and temporal awareness (Zep, `2501.13956`). CogMem aims to make the implementation of these dynamic strategies *customizable* via Lua scripting within the MMU and Reflection modules.

*   **Memory for Adaptation, Learning, and Goals:** We align with the goals of using LTM for self-evolution (`2410.15665`), embodied learning (MINDSTORES, `2501.19318`), and personalization (Second Me, `2503.08102`). CogMem's multi-entity structure allows personalization at both individual and group levels, and the reflection mechanism explicitly targets adaptation.

*   **Multi-Agent Systems:** While multi-agent LLM frameworks exist, specific mechanisms for fine-grained, entity-scoped shared memory within a cognitive architecture like CogMem are less explored. CogMem proposes a concrete architectural approach to manage shared state crucial for collaborative agent tasks.

*   **Scripting in AI Systems:** Embedding scripting languages (like Lua) for flexibility is common in game engines and other complex systems but less explicitly detailed in cognitive architecture proposals for LLMs. CogMem advocates for its use to balance core system stability (Go) with rapid customization (Lua).

*   **Broader Context and Critiques:** The AGI survey (`2501.03151`) positions memory, alongside embodiment, grounding, and causality, as a foundational requirement for achieving human-level general intelligence. Agent surveys (`2404.13501`, `2404.10890`) provide a broad overview of the state-of-the-art in LLM agents and memory mechanisms. Critically, the "Cognitive Mirage" paper (`2311.02982`) cautions against anthropomorphizing LLM agent capabilities based purely on performance, urging researchers to focus on functional benchmarks and avoid premature claims of cognitive equivalence. CogMem is designed with these perspectives in mind, aiming for functional improvements while maintaining careful framing of its capabilities.

**3. The CogMem Architecture**

CogMem is designed as a modular and extensible architecture with multi-tenancy and scriptiablity in mind. Its core philosophy is to separate concerns, manage memory explicitly across tiers, enable dynamic processing, and facilitate adaptation through reflection.

**(Figure 1: High-Level Diagram of CogMem Architecture - *Description below*)**
*(Imagine the previous diagram, but now explicitly showing an "Entity Context" input feeding into the Executive Controller and MMU. Perhaps indicate Lua script icons interacting with MMU, Reasoning, and Reflection modules.)*

**3.1 Components:**

*   **Perception Module:** Responsible for receiving and pre-processing input from the environment (e.g., user queries, sensor data placeholders). It identifies key information, entities, and potential user intent, source entity context (e.g user id, group id) and formatting it for downstream processing.
*   **Working Memory (WM) Manager:** Manages the LLM's active context window. This component is responsible for:
    *   Holding the immediate conversational context, current goals, and temporarily retrieved LTM information.
    *   Potentially prioritizing information within the context window (though sophisticated attention management within the context is often handled implicitly by the LLM itself).
    *   Interfacing with the MMU to request LTM retrieval or initiate storage of overflow information. Its state directly corresponds to the input provided to the LLM for reasoning.
*   **Long-Term Memory (LTM) Store:** The persistent memory repository. Key features:
    *   **Data Partitioning**: All stored data (vectors, graph nodes/edges, structured records) is tagged with an entity_id.
    *   **Access Control**: Data includes access level markers (e.g., private_to_user, shared_within_entity, public).
    *   **Hybrid Storage:** Combines multiple storage strategies for flexibility. Designed to potentially integrate (all capable of handling particned and accecc controlled data):
        *   A **vector database** for efficient semantic similarity search.
        *   A **structured database** (e.g., relational, key-value) for facts, entity profiles.
        *   A **temporal knowledge graph** (inspired by systems like Zep/Graphiti, `2501.13956`) for dynamically storing and relating conversational fragments, structured data, and their historical context.
    *   **Diverse Content:** Designed to store various types of information: semantic knowledge, episodic memories (past interactions, `(s, t, p, o)` tuples like MINDSTORES), procedural knowledge, user preferences, temporal relationships, and reflective insights.
    *   **CRUD Operations:** Supports standard Create, Read, Update, Delete operations managed via the MMU, adaptable to different backend types.
*   **Memory Management Unit (MMU):** The central coordinator for memory operations, acting as the interface between WM and LTM. Inspired by MemGPT's memory controller. Key functions:
    *   `encode_to_ltm(data, entity_context, access_level)`: Processes information (potentially summarizing or structuring it using the LLM) from WM deemed important for long-term storage and sends it to the LTM Store. Stores data tagged with the entity ID and specified access level.
    *   `retrieve_from_ltm(query, entity_context, options)`: Queries LTM, automatically filtering by entity_id and respecting access levels (retrieving private data for the specific user within the entity, plus shared data for the entity). Can leverage semantic search (vectors), structured queries (SQL/Cypher-like), or graph traversal depending on the query and available backends. Supports options for iterative refinement (ILM-TR inspired) and temporal filtering/reasoning (Zep inspired).
    *   `consolidate_ltm(insights, entity_context)`: Updates and refines LTM, within the entity context, based on new experiences or reflective insights. Could involve updating vector embeddings, modifying structured records, or adding/modifying nodes and edges in a knowledge graph, and potentially modifying shared data based on reflection. Consolidation rules can be defined in Lua scripts.
    *   `manage_wm_overflow(entity_context)`: Detects when WM (LLM context) is nearing its limit and decides what to offload from WM based on entity context and potential Lua-defined heuristics to LTM via `encode_to_ltm`.

*   **Reasoning Engine:** The core cognitive engine, typically leveraging the underlying LLM, 
    potentially using entity context information provided in the prompt. Complex reasoning strategies or sub-task decomposition could potentially invoke Lua scripts.
    *   **Decoupled Operation:** Receives input primarily from the WM Manager (which contains current context + retrieved LTM).
    *   **Structured Prompting:** Employs advanced prompting techniques (e.g., Chain-of-Thought [Wei et al., 2022], ReAct [Yao et al., 2022]) to perform reasoning, planning, and decision-making.
    *   **Memory Interaction:** Can explicitly request LTM retrieval via the MMU if needed information is missing from WM.
    *   **Output:** Generates thoughts, plans, and actions to be executed. Produces reasoning traces for the Reflection Module.
*   **Self-Reflection Module:** Enables meta-cognition and adaptation, inspired by SR-CIS
    (`2408.01970`) and the goal of self-evolution (`2410.15665`). Analyzes performance within an entity_context. Reflection triggers, analysis logic, and insight generation can be implemented or customized via Lua scripts.
    *   **Analysis:** Periodically or based on triggers (e.g., task failure, surprising outcome), analyzes recent interactions, reasoning traces, and task outcomes stored temporarily or in LTM.
    *   **Insight Generation:** Identifies patterns, errors, inconsistencies, or opportunities for improvement using the LLM.
    *   **Feedback Loop:** Generates reflective insights that can trigger `consolidate_ltm` operations in the MMU to update knowledge or strategies, or directly inform the Reasoning Engine for future tasks.
*   **Action Module:** Executes actions decided by the Reasoning Engine. This could involve generating a text response, calling an external API/tool, interacting with a simulated environment, or modifying the internal state, or interacting with entity-specific external systems
*   **Executive Controller:** The main control loop that orchestrates the interaction between all modules. It manages the overall agent lifecycle, maintains the current `entity_context` for each interaction, authentiactes requests/agents against entities, sequences module activation (e.g., Perception -> WM -> Reasoning -> Action), integrates the Reflection Module into the cycle and loads/manates relevant `Lua Scripts`.

**3.2 Key Interactions and Dynamics:**

*   **Tiered Memory Flow:** Input is perceived, placed in WM. The Reasoning Engine operates on WM. If necessary, the MMU retrieves relevant LTM chunks into WM. As WM fills, the MMU encodes less critical WM content into LTM. This dynamic flow, managed by the MMU, allows the agent to handle information exceeding the LLM's native context limit. All MMU interactions strictly adhere to the current `entity_context`.
*   **Dynamic Retrieval:** The `retrieve_from_ltm` function is crucial. It can be enhanced beyond simple vector search by querying graph structures for relationships, filtering by time (`2501.13956`), considering metadata (recency, importance), context-sensitivity (`2404.00573`), or employing iterative refinement loops (ILM-TR `2410.12859`). It is enhanced by entity filtering and potential Lua-scripted logic for relevance ranking or iterative query refinement within the entity's accessible memory scope.

*   **Reflective Adaptation Loop:** The Reflection Module provides a mechanism for learning and adaptation. By analyzing past performance (using data stored potentially in LTM), it can identify flawed reasoning patterns or outdated knowledge. Its output feeds back into the LTM via the `consolidate_ltm` function, allowing the agent to improve over time, addressing the goal of self-evolution (`2410.15665`). Reflection can also inform updates to graph structures. Operates within an entity context, allowing agents to adapt based on experiences specific to that user or group, potentially updating shared knowledge or strategies defined in Lua.

**3.3 Dynamic LTM:** The LTM is designed to be dynamic not just through updates but potentially through its structure. Using a temporal knowledge graph backend (`2501.13956`) allows the LTM to intrinsically represent evolving relationships and historical context, supporting more nuanced queries and reasoning compared to purely static or vector-based memories. Consolidation can involve intelligently merging nodes, updating relationships, or summarizing temporal event chains within the graph.

**3.4 Multi-Entity and Multi-Agent Support:**

*   **Entity Scoping:** CogMem enforces memory isolation between different entities (e.g., Company A's data is inaccessible to Company B). Each interaction is processed within the context of a single, identified entity.
*   **Shared Memory Segments:** Within a single entity's scope, memory can be designated as `shared_within_entity`. Multiple agents serving this entity (e.g., different chatbots for a company, collaborative agents working on a group project) can read and potentially write to this shared segment, enabling collaboration and consistent knowledge across agents.
*   **Access Control:** The `access_level` tag provides basic control. More complex models (e.g., role-based access, explicit sharing permissions designated by users) could be layered on top, potentially managed via Lua scripts interfacing with an external permission system.

**3.5 Scripting with Lua (`gopher-lua`):**

*   **Purpose:** To provide flexibility and enable rapid prototyping of complex or experimental logic without recompiling the core Go system.
*   **Integration:** The Go core loads and executes Lua scripts at predefined extension points within modules like the MMU (e.g., custom retrieval filters, scoring functions), Reflection Module (e.g., analysis patterns, insight formatting), and potentially the Reasoning Engine (e.g., task-specific sub-routines).
*   **Core vs. Script:** Core functionalities like database drivers, networking, basic data structures, and the main architectural loop remain in efficient, compiled Go code. Higher-level logic, business rules, or experimental algorithms can reside in Lua scripts.
*   **Benefits:** Faster iteration cycles for logic changes, easier customization for specific deployments, potential for user-defined agent behaviors.

**4. Implementation Details (CogMem Library)**

While this paper proposes the architecture, we envision its realization as an open-source **Golang/Python library**.

*   **Library Goal:** To provide researchers and developers with a flexible, efficient, and extensible multi-tenant framework for building CogMem-based agents, leveraging Go's strengths in concurrency and performance (Go core), with scriptable framework for CogMem agents via `gopher-lua`.
*   **Core Components:** The library would offer core Go packages and interfaces (and corresponding Python classes) representing each module (e.g., `CogMemAgent`, `WorkingMemoryManager`, `LTMStore`, `MMU`, `ReasoningEngine`, `ReflectionModule`, `ActionModule`). Go's strong typing and interface system would define clear contracts between modules, promoting modularity and testability. Data structures will include explicit fields for entity_id and access_level. Mechanisms for managing entity contexts during requests will be central. Integration with gopher-lua or a similar robust Lua interpreter for Go will be a core feature.
*   **Interfaces:** Well-defined Go interfaces are critical for modularity. For instance, the `LTMStore` interface would define methods for CRUD operations, allowing different backend databases (vector DBs like Weaviate/Milvus accessible via Go clients, graph DBs like Dgraph/Neo4j **or potentially interacting with systems like Zep if APIs allow**, key-value stores like Redis/etcd) to be plugged in. Similar interfaces would exist for reasoning engines and action modules. LTM Store interfaces will require methods to handle entity IDs and access levels during CRUD operations. MMU interfaces will accept entity_context parameters.
*   **Technology Stack:** Primarily **Golang**, capitalizing on its concurrency model for potentially parallelizing memory operations, reflection processes, or handling multiple agent interactions efficiently. Alternatively, **Python** implementations would be provided for integration with existing ML ecosystems. Key dependencies would include standard LLM client libraries (Go bindings for OpenAI, Anthropic, etc., or libraries for interacting with local models), Go clients for selected vector/graph/KV databases, and standard Go libraries for networking and data handling.gopher-lua for scripting. Alternatively, Python implementations/wrappers. 
*   **Configuration:** Users should be able to easily configure the agent (e.g., via YAML or JSON files) by selecting the LLM provider/model, LTM backends (specifying type: vector, graph, etc.), specific retrieval strategies, reflection triggers, prompting techniques, paths to Lua scripts implementing customizable logic for various modules regardless of whether they are using the Go or Python version of the library. 

**5. Proposed Evaluation**

Evaluating CogMem requires assessing its core memory functions plus its multi-tenant and scripting capabilities.

*   **Methodology:** Comparative evaluation against baselines on tasks testing core memory, collaboration, isolation, and adaptation.
*   **Tasks:**
    *   *Core Memory Tasks:* Long Conversation QA, Complex Instruction Following, Temporal Reasoning (using DMR/LongMemEval benchmarks where applicable).
    *   ***Multi-Entity/Agent Tasks:***
        *   *Cross-Agent Collaboration:* Two agents serving the same group entity use shared LTM to complete a task neither could do alone.
        *   *Entity Isolation Test:* Verify that agents serving different entities cannot access each other's private or shared data.
        *   *Personalization vs. Group Knowledge:* Agent correctly utilizes both `private_to_user` preferences and `shared_within_entity` knowledge.
        *   *Concurrent Access:* Test system behavior with multiple agents concurrently reading/writing to shared memory segments.
    *   *Adaptation Tasks:* Error Correction via Reflection, Adaptation to changing entity needs (potentially using Lua-defined reflection logic).
    *   *(Optional) Scripting Flexibility:* Demonstrate implementing different retrieval strategies or reflection rules using Lua scripts and measure ease of modification versus baseline.
*   **Metrics:** Task success rates, F1 recall, isolation success rate, collaboration efficiency, adaptation speed, latency (including Lua execution overhead), token usage. Qualitative assessment of coherence, reasoning, and script usability.
*   **Baselines:** Vanilla LLM, RAG, MemGPT, SR-CIS, Zep (where applicable and available). Potentially a version of CogMem with Lua scripting disabled for performance comparison.

**6. Discussion**

CogMem offers a comprehensive approach by integrating advanced memory concepts with crucial features for practical deployment.

*   **Advantages:** Synthesis of modularity, tiered memory, dynamic processes, and reflection. **Crucially adds multi-entity awareness and shared memory for collaboration, essential for many real-world use cases.** **Embedded Lua scripting provides significant flexibility, enabling rapid customization and experimentation without modifying the core system.** The primary Go implementation targets performance and concurrency.
*   **Limitations:** Increased architectural complexity due to multi-tenancy and scripting. Managing access control policies effectively can be challenging. Concurrent access to shared memory requires careful handling (e.g., locking mechanisms or conflict resolution strategies). **Embedding a scripting language introduces potential performance overhead and security considerations (sandboxing Lua execution will be critical).** Tuning interactions between Go core and Lua scripts requires care.
*   **Memory Spectrum and Cognitive Fidelity:** CogMem focuses on architectural/external memory but provides hooks (potentially via Lua) to interact with other types. Its multi-entity nature adds a layer of contextual grounding often missing. We maintain a focus on functional capabilities over cognitive claims, adhering to the "Cognitive Mirage" critique.
*   **Towards More Capable AI:** Providing robust, contextualized memory and adaptation mechanisms addresses key limitations of current LLMs. Multi-agent collaboration via shared memory unlocks new possibilities. These features contribute practical steps towards more capable, deployable AI systems, aligning with broader goals like self-evolution and AGI foundations.
*   **Future Work:** Developing sophisticated access control models; exploring efficient concurrency control for shared memory; optimizing Go-Lua interoperation; extending scripting capabilities; investigating automatic generation or refinement of Lua scripts via reflection; deeper integration with causal reasoning and embodiment frameworks.

**7. Conclusion**

We have proposed CogMem, a modular cognitive architecture designed for building advanced LLM agents. It uniquely integrates established concepts like tiered memory, dynamic processing, and reflection with novel features crucial for practical application: **multi-entity awareness, multi-agent shared memory, and embedded Lua scripting for customization.** By providing a principled way to manage memory contextually for different users and groups, enabling collaboration between agents, and offering unparalleled flexibility through scripting, CogMem addresses key limitations of existing systems. While implementation presents challenges, CogMem offers a promising blueprint for more coherent, adaptive, collaborative, and ultimately more capable LLM agents. We plan to develop CogMem as an **open-source library, prioritizing a Go implementation for performance and concurrency, while providing placeholder for future Python interfaces,** to spur further innovation in this vital domain.

**8. References**

*(Note: ArXiv IDs are used where confirmed or provided in abstracts. Standard citation format should be applied)*

*   [Brown et al., 2020] Brown, T. B., Mann, B., Ryder, N., et al. (2020). Language Models are Few-Shot Learners. *arXiv preprint arXiv:2005.14165*.
*   [`2310.08998`] Packer, C., et al. (2023). MemGPT: Towards LLMs as Operating Systems. *arXiv preprint arXiv:2310.08998*.
*   [`2310.08560`] Wang, L., et al. (2023). Cognitive Architectures for Language Agents. *arXiv preprint arXiv:2310.08560*.
*   [`2309.02427`] Chen, W., et al. (2023). MemoryBank: Enhancing Large Language Models with Long-Term Memory. *arXiv preprint arXiv:2309.02427*.
*   [`2404.10890`] Zhang, M., et al. (2024). LLM Agents as Operating Systems: A Survey. *arXiv preprint arXiv:2404.10890*.
*   [`2504.02441`] Shan, L., Luo, S., Zhu, Z., Yuan, Y., & Wu, Y. (2025). Cognitive Memory in Large Language Models. *arXiv preprint arXiv:2504.02441*. (Cited based on provided abstract)
*   [`2410.15665`] Jiang, X., Li, F., Zhao, H., et al. (2024). Long Term Memory: The Foundation of AI Self-Evolution. *arXiv preprint arXiv:2410.15665*. (Cited based on provided abstract)
*   [`2503.08102`] Wei, J., Ying, X., Gao, T., Bao, F., Tao, F., & Shang, J. (2025). AI-native Memory 2.0: Second Me. *arXiv preprint arXiv:2503.08102*. (Cited based on provided abstract)
*   [`2404.13501`] Chen, X., et al. (2024). A Survey on the Memory Mechanism of Large Language Model based Agents. *arXiv preprint arXiv:2404.13501*.
*   [`2408.09176`] Yang, K., et al. (2024). Cognitive LLMs: Towards Integrating Cognitive Architectures and Large Language Models for Manufacturing Decision-making. *arXiv preprint arXiv:2408.09176*.
*   [`2501.19318`] Chari, A., Reddy, S., Tiwari, A., Lian, R., & Zhou, B. (2025). MINDSTORES: Memory-Informed Neural Decision Synthesis for Task-Oriented Reinforcement in Embodied Systems. *arXiv preprint arXiv:2501.19318*. (Cited based on provided abstract)
*   [`2404.00573`] Lei, Z., et al. (2024). "My agent understands me better": Integrating Dynamic Human-like Memory Recall and Consolidation in LLM-Based Agents. *arXiv preprint arXiv:2404.00573*.
*   [`2501.03151`] Mumuni, A., & Mumuni, F. (2025). Large language models for artificial general intelligence (AGI): A survey of foundational principles and approaches. *arXiv preprint arXiv:2501.03151*. (Cited based on provided abstract)
*   [`2408.01970`] Zhou, Y., et al. (2024). SR-CIS: Self-Reflective Incremental System with Decoupled Memory and Reasoning. *arXiv preprint arXiv:2408.01970*.
*   [`2410.12859`] Tang, Y., Xu, Y., Yan, N., & Mortazavi, M. (2024). Enhancing Long Context Performance in LLMs Through Inner Loop Query Mechanism. *arXiv preprint arXiv:2410.12859*. (Cited based on provided abstract)
*   [`2501.13956`] Rasmussen, P., Paliychuk, P., Beauvais, T., Ryan, J., & Chalef, D. (2025). Zep: A Temporal Knowledge Graph Architecture for Agent Memory. *arXiv preprint arXiv:2501.13956*. (Cited based on provided abstract)
*   [`2311.02982`] Jones, C., & Steinhardt, J. (2023). Cognitive Mirage: A Review of Anthropomorphism in Large Language Model-Based Agent Research. *arXiv preprint arXiv:2311.02982*.
*   [Lewis et al., 2020] Lewis, P., Perez, E., Piktus, A., et al. (2020). Retrieval-Augmented Generation for Knowledge-Intensive NLP Tasks. *arXiv preprint arXiv:2005.11401*.
*   [Mialon et al., 2023] Mialon, G., et al. (2023). Augmented Language Models: a Survey. *arXiv preprint arXiv:2302.07842*.
*   [Wang et al., 2023] Wang, L., et al. (2023). A Survey on Large Language Model based Autonomous Agents. *arXiv preprint arXiv:2308.11432*.
*   [Wei et al., 2022] Wei, J., Wang, X., Schuurmans, D., et al. (2022). Chain-of-Thought Prompting Elicits Reasoning in Large Language Models. *arXiv preprint arXiv:2201.11903*.
*   [Xi et al., 2023] Xi, Z., Chen, W., Guo, X., et al. (2023). The Rise and Potential of Large Language Model Based Agents: A Survey. *arXiv preprint arXiv:2309.07864*.
*   [Yao et al., 2022] Yao, S., Zhao, J., Yu, D., et al. (2022). ReAct: Synergizing Reasoning and Acting in Language Models. *arXiv preprint arXiv:2210.03629*.

---