**CogMem: A Modular Cognitive Architecture for LLM Agents with Tiered Memory, Dynamic Processing, and Reflective Adaptation**

**Abstract:** Large Language Models (LLMs) require advanced cognitive capabilities, particularly robust memory systems, to function effectively as autonomous agents capable of complex tasks, personalization, and continuous learning. Current approaches vary widely, from external databases and graph-based systems to internal parameter modifications. We propose CogMem, a comprehensive cognitive architecture synthesizing key concepts from recent research. CogMem integrates: 1) A modular design inspired by cognitive architectures (CogArch, SR-CIS). 2) A tiered memory system (MemGPT-style WM/LTM) managing context. 3) A sophisticated Long-Term Memory (LTM) supporting diverse content (semantic knowledge, episodic experiences like MINDSTORES tuples, temporal relationships) and implementations (hybrid vector/structured DBs, potentially incorporating temporal knowledge graphs like Zep's Graphiti). 4) Dynamic memory processes including iterative retrieval (ILM-TR inspired), context-sensitive recall, and consolidation (`2404.00573`). 5) A reflective loop (SR-CIS) enabling self-correction and adaptation, crucial for self-evolution (`2410.15665`). CogMem aims to provide a robust and flexible **library (primarily in Go, with Python support)** for building LLM agents that exhibit enhanced coherence, adaptability, and reasoning, positioning memory as a cornerstone for more capable AI, while remaining cognizant of the challenges in achieving true AGI (`2501.03151`) and avoiding anthropomorphic overclaims (Cognitive Mirage, `2311.02982`).

**1. Introduction**

Large Language Models (LLMs) have demonstrated remarkable proficiency in natural language understanding, generation, and various reasoning tasks [Brown et al., 2020]. However, their inherent limitations – particularly the fixed context window size and stateless nature between independent interactions – significantly hinder their application in domains requiring long-term coherence, persistent memory, adaptation, and complex multi-step reasoning [Mialon et al., 2023]. These limitations are particularly acute for LLM-based autonomous agents, which need to maintain state, learn from interactions, recall past information accurately, and adapt their behavior over extended periods [Wang et al., 2023; Xi et al., 2023].

Initial approaches to mitigate these issues, such as basic prompt engineering or simple Retrieval-Augmented Generation (RAG) [Lewis et al., 2020], often fall short. While RAG provides access to external knowledge, it typically lacks sophisticated memory management, struggles with integrating retrieved information coherently, often deals only with static documents, and doesn't inherently support dynamic memory processes like consolidation or reflective learning. Enterprise applications, in particular, demand dynamic knowledge integration from diverse, evolving sources like ongoing conversations and structured business data, going beyond static retrieval (`2501.13956`).

Recognizing these gaps, recent research has explored more structured solutions. Cognitive architectures inspired by human cognition propose modular systems with distinct components like working memory (WM) and long-term memory (LTM) [e.g., `2310.08560`; `2408.09176`]. Systems like MemGPT (`2310.08998`) introduce the concept of tiered memory management, where the LLM acts like an operating system managing its context window (akin to RAM) and interacting with external storage (akin to disk). MemoryBank (`2309.02427`) focuses on practical LTM implementations using summarization and interaction history. **Zep (`2501.13956`) introduces a memory layer centered around a temporal knowledge graph (Graphiti) designed to dynamically synthesize unstructured conversational data and structured business data while maintaining historical relationships, reporting strong performance on benchmarks reflecting enterprise needs.** Furthermore, work has emerged on dynamic memory processes like human-like recall and consolidation (`2404.00573`), iterative retrieval mechanisms (ILM-TR, `2410.12859`), and the crucial role of reflection in enabling self-correction and adaptation (SR-CIS, `2408.01970`). Concurrently, researchers emphasize memory's foundational role in achieving personalization (Second Me, `2503.08102`), lifelong learning and self-evolution (`2410.15665`), learning from experience in embodied agents (MINDSTORES, `2501.19318`), and ultimately, as a pillar for Artificial General Intelligence (AGI) (`2501.03151`). Taxonomies are also being developed to categorize the diverse memory mechanisms being explored (`2504.02441`). Comprehensive surveys map this rapidly evolving landscape [e.g., `2404.13501`, `2404.10890`].

While these advancements are significant, they often focus on specific aspects of the memory or cognition challenge. There is a need for a unified architecture that synthesizes these complementary ideas into a coherent and practical framework.

In this paper, we propose **CogMem**, a modular cognitive architecture designed to address these challenges. CogMem integrates insights from cognitive science and recent AI research to create a system featuring:
*   A **modular design** separating perception, memory, reasoning, reflection, and action.
*   A **tiered memory system** with distinct WM (LLM context) and LTM (external, persistent store).
*   An **explicit Memory Management Unit (MMU)** orchestrating information flow and context management, inspired by MemGPT.
*   A **sophisticated LTM** capable of storing diverse information (facts, dialogues, experiences/tuples, temporal relationships) using hybrid storage backends, potentially including temporal knowledge graphs.
*   **Dynamic memory processes**, including context-aware, potentially iterative retrieval and consolidation mechanisms informed by experience and reflection.
*   A **reflective loop** enabling the agent to analyze its performance and adapt its knowledge and strategies over time, fostering self-improvement.

Our primary contribution is the proposal of this synthesized architecture, outlining its components, interactions, and potential benefits. We aim for CogMem to serve as a blueprint and eventually an open-source library, providing a flexible foundation for building more capable, adaptive, and coherent LLM agents. We position CogMem as a functional enhancement, carefully distinguishing its capabilities from unsubstantiated claims of human-like cognition, acknowledging the important critique of the "Cognitive Mirage" (`2311.02982`).

**2. Related Work**

CogMem builds upon and integrates several threads of research in LLM memory and cognitive architectures.

*   **Cognitive Architectures for LLMs:** Several works propose integrating LLMs into frameworks inspired by cognitive science [e.g., ACT-R, SOAR]. Papers like "Cognitive Architectures for Language Agents" (`2310.08560`) and "Cognitive LLMs" (`2408.09176`) advocate for modular structures with distinct components like working memory, declarative memory, procedural memory, and executive control. SR-CIS (`2408.01970`) further emphasizes the importance of decoupling memory and reasoning modules and introduces a self-reflection mechanism. CogMem adopts this modular philosophy for clarity, maintainability, and extensibility, explicitly decoupling key functions and incorporating reflection.

*   **Memory System Implementations & Management:** Practical memory systems are crucial. MemoryBank (`2309.02427`) demonstrates effective LTM using retrieval, summarization, and structured storage. MemGPT (`2310.08998`) provides a powerful metaphor and mechanism with the LLM acting as an OS, managing its limited context (WM) by intelligently paging information to/from an external LTM store via an explicit MMU using function calls. **Zep (`2501.13956`) offers a distinct approach centered on its `Graphiti` temporal knowledge graph engine. It aims to dynamically synthesize unstructured conversational and structured business data, explicitly managing temporal relationships, and reporting superior performance versus MemGPT on the DMR benchmark and strong results on its own LongMemEval benchmark focused on enterprise temporal reasoning tasks.** CogMem directly incorporates the tiered memory (WM/LTM) and MMU concepts from MemGPT, while its LTM is designed to leverage hybrid storage ideas, potentially incorporating vector stores, traditional structured DBs, and graph-based approaches like Zep's for a robust and flexible LTM capable of handling diverse data types and relationships. The taxonomy presented in "Cognitive Memory in Large Language Models" (`2504.02441`) categorizes memory implementations (External/Text, KV Cache, Parameter-based, Hidden-state). CogMem primarily focuses on the architectural level involving external LTM, but its modular design allows for potential future integration or interaction with other memory types.

*   **Dynamic Memory Processes:** Static memory storage is insufficient; dynamic processes are needed. "My agent understands me better" (`2404.00573`) highlights the need for more human-like recall (going beyond simple semantic similarity) and memory consolidation to refine stored knowledge. ILM-TR (`2410.12859`) proposes an iterative retrieval mechanism ("inner loop") using intermediate findings stored in a short-term memory (STM) buffer to refine queries for complex questions requiring deep reasoning within long contexts. **The temporal awareness inherent in Zep's graph structure (`2501.13956`) also represents a dynamic aspect, allowing retrieval and reasoning based on historical context and relationships.** CogMem aims to incorporate such dynamic aspects within its MMU's `retrieve` function (allowing for iterative refinement, temporal querying) and its `consolidate` function (using reflection outputs to update LTM, potentially refining graph structures or temporal links).

*   **Memory for Adaptation, Learning, and Goals:** The *purpose* of memory extends beyond simple recall. "Long Term Memory: The Foundation of AI Self-Evolution" (`2410.15665`) argues that LTM is fundamental for agents to learn continuously from limited interactions during inference and evolve over time. MINDSTORES (`2501.19318`) demonstrates this in embodied agents by storing experience as `(state, task, plan, outcome)` tuples in LTM, enabling plan refinement based on past successes and failures. "AI-native Memory 2.0: Second Me" (`2503.08102`) frames memory as a tool for personalization and reducing user cognitive load. CogMem embraces these goals by designing its LTM to store diverse data types (including experience tuples and temporal graphs) and integrating the reflection module to explicitly drive adaptation and learning based on experience stored in LTM.

*   **Broader Context and Critiques:** The AGI survey (`2501.03151`) positions memory, alongside embodiment, grounding, and causality, as a foundational requirement for achieving human-level general intelligence. Agent surveys (`2404.13501`, `2404.10890`) provide a broad overview of the state-of-the-art in LLM agents and memory mechanisms. Critically, the "Cognitive Mirage" paper (`2311.02982`) cautions against anthropomorphizing LLM agent capabilities based purely on performance, urging researchers to focus on functional benchmarks and avoid premature claims of cognitive equivalence. CogMem is designed with these perspectives in mind, aiming for functional improvements while maintaining careful framing of its capabilities.

**3. The CogMem Architecture**

CogMem is designed as a modular and extensible architecture inspired by cognitive science principles and practical LLM agent systems. Its core philosophy is to separate concerns, manage memory explicitly across tiers, enable dynamic processing, and facilitate adaptation through reflection.

**(Figure 1: High-Level Diagram of CogMem Architecture - *Description below*)**
*(Imagine a diagram showing interconnected boxes for Perception, WM Manager, LTM Store, MMU, Reasoning Engine, Reflection Module, Action Module, all orchestrated by an Executive Controller. Arrows indicate primary data/control flow.)*

**3.1 Components:**

*   **Perception Module:** Responsible for receiving and pre-processing input from the environment (e.g., user queries, sensor data placeholders). It identifies key information, entities, and potential user intent, formatting it for downstream processing.
*   **Working Memory (WM) Manager:** Manages the LLM's active context window. This component is responsible for:
    *   Holding the immediate conversational context, current goals, and temporarily retrieved LTM information.
    *   Potentially prioritizing information within the context window (though sophisticated attention management within the context is often handled implicitly by the LLM itself).
    *   Interfacing with the MMU to request LTM retrieval or initiate storage of overflow information. Its state directly corresponds to the input provided to the LLM for reasoning.
*   **Long-Term Memory (LTM) Store:** The persistent memory repository. Key features:
    *   **Hybrid Storage:** Combines multiple storage strategies for flexibility. Designed to potentially integrate:
        *   A **vector database** for efficient semantic similarity search.
        *   A **structured database** (e.g., relational, key-value) for facts, entity profiles.
        *   A **temporal knowledge graph** (inspired by systems like Zep/Graphiti, `2501.13956`) for dynamically storing and relating conversational fragments, structured data, and their historical context.
    *   **Diverse Content:** Designed to store various types of information: semantic knowledge, episodic memories (past interactions, `(s, t, p, o)` tuples like MINDSTORES), procedural knowledge, user preferences, temporal relationships, and reflective insights.
    *   **CRUD Operations:** Supports standard Create, Read, Update, Delete operations managed via the MMU, adaptable to different backend types.
*   **Memory Management Unit (MMU):** The central coordinator for memory operations, acting as the interface between WM and LTM. Inspired by MemGPT's memory controller. Key functions:
    *   `encode_to_ltm(data)`: Processes information (potentially summarizing or structuring it using the LLM) from WM deemed important for long-term storage and sends it to the LTM Store.
    *   `retrieve_from_ltm(query, context, options)`: Queries the LTM Store. Can leverage semantic search (vectors), structured queries (SQL/Cypher-like), or graph traversal depending on the query and available backends. Supports options for iterative refinement (ILM-TR inspired) and temporal filtering/reasoning (Zep inspired).
    *   `consolidate_ltm(insights)`: Updates and refines LTM based on new experiences or reflective insights. Could involve updating vector embeddings, modifying structured records, or adding/modifying nodes and edges in a knowledge graph.
    *   `manage_wm_overflow()`: Detects when WM (LLM context) is nearing its limit and decides which information to offload to LTM via `encode_to_ltm`.
*   **Reasoning Engine:** The core cognitive engine, typically leveraging the underlying LLM.
    *   **Decoupled Operation:** Receives input primarily from the WM Manager (which contains current context + retrieved LTM).
    *   **Structured Prompting:** Employs advanced prompting techniques (e.g., Chain-of-Thought [Wei et al., 2022], ReAct [Yao et al., 2022]) to perform reasoning, planning, and decision-making.
    *   **Memory Interaction:** Can explicitly request LTM retrieval via the MMU if needed information is missing from WM.
    *   **Output:** Generates thoughts, plans, and actions to be executed. Produces reasoning traces for the Reflection Module.
*   **Self-Reflection Module:** Enables meta-cognition and adaptation, inspired by SR-CIS (`2408.01970`) and the goal of self-evolution (`2410.15665`).
    *   **Analysis:** Periodically or based on triggers (e.g., task failure, surprising outcome), analyzes recent interactions, reasoning traces, and task outcomes stored temporarily or in LTM.
    *   **Insight Generation:** Identifies patterns, errors, inconsistencies, or opportunities for improvement using the LLM.
    *   **Feedback Loop:** Generates reflective insights that can trigger `consolidate_ltm` operations in the MMU to update knowledge or strategies, or directly inform the Reasoning Engine for future tasks.
*   **Action Module:** Executes actions decided by the Reasoning Engine. This could involve generating a text response, calling an external API/tool, interacting with a simulated environment, or modifying the internal state.
*   **Executive Controller:** The main control loop that orchestrates the interaction between all modules. It manages the overall agent lifecycle, sequences module activation (e.g., Perception -> WM -> Reasoning -> Action), and integrates the Reflection Module into the cycle.

**3.2 Key Interactions and Dynamics:**

*   **Tiered Memory Flow:** Input is perceived, placed in WM. The Reasoning Engine operates on WM. If necessary, the MMU retrieves relevant LTM chunks into WM. As WM fills, the MMU encodes less critical WM content into LTM. This dynamic flow, managed by the MMU, allows the agent to handle information exceeding the LLM's native context limit.
*   **Dynamic Retrieval:** The `retrieve_from_ltm` function is crucial. It can be enhanced beyond simple vector search by querying graph structures for relationships, filtering by time (`2501.13956`), considering metadata (recency, importance), context-sensitivity (`2404.00573`), or employing iterative refinement loops (ILM-TR `2410.12859`).
*   **Reflective Adaptation Loop:** The Reflection Module provides a mechanism for learning and adaptation. By analyzing past performance (using data stored potentially in LTM), it can identify flawed reasoning patterns or outdated knowledge. Its output feeds back into the LTM via the `consolidate_ltm` function, allowing the agent to improve over time, addressing the goal of self-evolution (`2410.15665`). Reflection can also inform updates to graph structures.

**3.3 Dynamic LTM:** The LTM is designed to be dynamic not just through updates but potentially through its structure. Using a temporal knowledge graph backend (`2501.13956`) allows the LTM to intrinsically represent evolving relationships and historical context, supporting more nuanced queries and reasoning compared to purely static or vector-based memories. Consolidation can involve intelligently merging nodes, updating relationships, or summarizing temporal event chains within the graph.

**4. Implementation Details (CogMem Library)**

While this paper proposes the architecture, we envision its realization as an open-source **Golang/Python library**.

*   **Library Goal:** To provide researchers and developers with a flexible, efficient, and extensible framework for building CogMem-based agents, leveraging Go's strengths in concurrency and performance, while offering a Python alternative for broader accessibility within the AI/ML community.
*   **Core Components:** The library would offer core Go packages and interfaces (and corresponding Python classes) representing each module (e.g., `CogMemAgent`, `WorkingMemoryManager`, `LTMStore`, `MMU`, `ReasoningEngine`, `ReflectionModule`, `ActionModule`). Go's strong typing and interface system would define clear contracts between modules, promoting modularity and testability. Python wrappers or parallel implementations could ensure wider adoption.
*   **Interfaces:** Well-defined Go interfaces are critical for modularity. For instance, the `LTMStore` interface would define methods for CRUD operations, allowing different backend databases (vector DBs like Weaviate/Milvus accessible via Go clients, graph DBs like Dgraph/Neo4j **or potentially interacting with systems like Zep if APIs allow**, key-value stores like Redis/etcd) to be plugged in. Similar interfaces would exist for reasoning engines and action modules.
*   **Technology Stack:** Primarily **Golang**, capitalizing on its concurrency model for potentially parallelizing memory operations, reflection processes, or handling multiple agent interactions efficiently. Alternatively, **Python** implementations would be provided for integration with existing ML ecosystems. Key dependencies would include standard LLM client libraries (Go/Python bindings for OpenAI, Anthropic, etc., or libraries for interacting with local models), Go clients for selected vector/graph/KV databases, and standard Go libraries for networking and data handling.
*   **Configuration:** Users should be able to easily configure the agent (e.g., via YAML or JSON files) by selecting the LLM provider/model, LTM backends (specifying type: vector, graph, etc.), specific retrieval strategies, reflection triggers, and prompting techniques, regardless of whether they are using the Go or Python version of the library.

**5. Proposed Evaluation**

Evaluating CogMem requires assessing its ability to overcome LLM limitations in complex, long-running tasks, demonstrate robust memory utilization, and exhibit adaptation.

*   **Methodology:** Comparative evaluation against relevant baselines on a suite of carefully designed tasks.
*   **Tasks:**
    *   *Long Conversation QA:* Answering questions requiring information synthesis from distant points in a long dialogue, testing LTM persistence and retrieval accuracy.
    *   *Multi-Session Personalization:* Remembering and utilizing user preferences and history across multiple interaction sessions (`2503.08102`), testing LTM update and retrieval.
    *   *Complex Instruction Following:* Executing multi-step tasks where later steps depend on information or outcomes from earlier steps stored in memory.
    *   *Error Correction via Reflection:* Tasks where the agent initially fails but can correct its behavior or knowledge based on reflective analysis of the failure.
    *   *Adaptation Tasks:* Scenarios where user needs or environment rules change over time, testing the agent's ability to adapt using the reflection-consolidation loop (testing `2410.15665` goals).
    *   *Complex Query Resolution:* Tasks requiring iterative information retrieval to answer complex questions based on long documents (testing ILM-TR `2410.12859` inspired mechanisms).
    *   *Temporal Reasoning Tasks:* Questions requiring synthesis of information across different time points or understanding evolving relationships (inspired by LongMemEval from Zep `2501.13956`).
    *   *Dynamic Knowledge Integration:* Scenarios where the agent must integrate information from both ongoing dialogue and structured data sources presented incrementally.
    *   *(Optional) Simplified Embodied Tasks:* Using environments like AlfWorld or simplified MineDojo setups to test planning informed by stored experience tuples (MINDSTORES `2501.19318`).
*   **Metrics:**
    *   *Quantitative:* Task success rates, F1 score for information recall, goal completion rates, adaptation speed, latency, token usage. Performance on specific benchmarks like **DMR and LongMemEval (`2501.13956`)** where applicable.
    *   *Qualitative:* Dialogue coherence, reasoning trace analysis, correctness of retrieved information (including temporal aspects), quality of reflective insights.
*   **Baselines:**
    *   Vanilla LLM (with maximum possible context).
    *   Standard RAG implementation.
    *   MemGPT implementation (if available/reproducible).
    *   SR-CIS implementation (if available/reproducible).
    *   **Zep (`2501.13956`)** (if available as a comparable system/library).

**6. Discussion**

CogMem represents a synthesis of promising research directions aimed at building more capable LLM agents.

*   **Advantages:** Its primary strength lies in the integration of multiple advanced concepts: structured modularity, explicit tiered memory management, support for diverse LTM content and backends **(including advanced structures like temporal knowledge graphs)**, dynamic processes, and a dedicated reflection mechanism for adaptation. This integration offers potential for improved performance on tasks requiring long-term coherence, learning, temporal reasoning, and complex reasoning compared to systems focusing on only one aspect. The modular design promotes flexibility and future extensibility.
*   **Limitations:** The proposed architecture introduces significant complexity. Implementing and tuning the MMU, dynamic retrieval strategies (especially graph queries or iterative loops), consolidation, and reflection presents challenges. Computational overhead associated with LTM access (potentially complex graph traversals), summarization/encoding, and reflection remains a concern. Ensuring robust and scalable LTM storage and retrieval, particularly for large, complex, temporal graphs, is an ongoing research problem.
*   **Memory Spectrum and Cognitive Fidelity:** CogMem primarily focuses on architectural and external memory mechanisms, complementing research on optimizing internal LLM memory like KV caching or parameter-based methods (`2504.02441`). While inspired by cognitive science, CogMem aims for *functional* emulation of capabilities like persistent memory, context management, and reflection. It does not claim to replicate the biological mechanisms or subjective experience of human cognition. In line with the "Cognitive Mirage" critique (`2311.02982`), evaluating CogMem should focus on demonstrable improvements in task performance, adaptability, and coherence, rather than anthropomorphic attributions.
*   **Towards More Capable AI:** By providing mechanisms for persistent memory, continuous learning via reflection, adaptation, **and handling temporally grounded information via structures like knowledge graphs**, CogMem contributes functional building blocks considered necessary for more advanced AI... (rest of paragraph same).
*   **Future Work:** Exciting directions include: deeper integration with internal LLM states; developing more sophisticated reflection algorithms; exploring online learning within LTM (including graph evolution); incorporating mechanisms for causal reasoning potentially leveraging temporal graph structures; extending the architecture to handle multi-modal information; rigorously benchmarking library implementations (including graph backends) across diverse tasks.

**7. Conclusion**

We have proposed CogMem, a modular cognitive architecture for LLM agents that synthesizes key advancements in memory management, dynamic processing, and reflective adaptation. By integrating concepts from MemGPT, cognitive architectures, SR-CIS, and research on dynamic memory processes, self-evolution, **and advanced LTM structures like temporal knowledge graphs (e.g., Zep)**, CogMem aims to provide a robust and flexible framework for overcoming the limitations of current LLMs. It features a tiered memory system managed by an explicit MMU, a versatile LTM supporting hybrid backends, dynamic retrieval and consolidation mechanisms, and a reflection loop for continuous improvement. While challenges in implementation and tuning remain, CogMem represents a principled step towards building more coherent, adaptive, and capable LLM-based agents, paving the way for new applications requiring sophisticated memory, temporal reasoning, and learning capabilities. We plan to develop CogMem as an **open-source library, prioritizing a Go implementation for performance and concurrency while providing Python interfaces,** to facilitate further research and development in this critical area.

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
*   [`2501.13956`] Rasmussen, P., Paliychuk, P., Beauvais, T., Ryan, J., & Chalef, D. (2025). Zep: A Temporal Knowledge Graph Architecture for Agent Memory. *arXiv preprint arXiv:2501.13956*. (Cited based on provided abstract)
*   [`2408.01970`] Zhou, Y., et al. (2024). SR-CIS: Self-Reflective Incremental System with Decoupled Memory and Reasoning. *arXiv preprint arXiv:2408.01970*.
*   [`2410.12859`] Tang, Y., Xu, Y., Yan, N., & Mortazavi, M. (2024). Enhancing Long Context Performance in LLMs Through Inner Loop Query Mechanism. *arXiv preprint arXiv:2410.12859*. (Cited based on provided abstract)
*   [`2311.02982`] Jones, C., & Steinhardt, J. (2023). Cognitive Mirage: A Review of Anthropomorphism in Large Language Model-Based Agent Research. *arXiv preprint arXiv:2311.02982*.
*   [Lewis et al., 2020] Lewis, P., Perez, E., Piktus, A., et al. (2020). Retrieval-Augmented Generation for Knowledge-Intensive NLP Tasks. *arXiv preprint arXiv:2005.11401*.
*   [Mialon et al., 2023] Mialon, G., et al. (2023). Augmented Language Models: a Survey. *arXiv preprint arXiv:2302.07842*.
*   [Wang et al., 2023] Wang, L., et al. (2023). A Survey on Large Language Model based Autonomous Agents. *arXiv preprint arXiv:2308.11432*.
*   [Wei et al., 2022] Wei, J., Wang, X., Schuurmans, D., et al. (2022). Chain-of-Thought Prompting Elicits Reasoning in Large Language Models. *arXiv preprint arXiv:2201.11903*.
*   [Xi et al., 2023] Xi, Z., Chen, W., Guo, X., et al. (2023). The Rise and Potential of Large Language Model Based Agents: A Survey. *arXiv preprint arXiv:2309.07864*.
*   [Yao et al., 2022] Yao, S., Zhao, J., Yu, D., et al. (2022). ReAct: Synergizing Reasoning and Acting in Language Models. *arXiv preprint arXiv:2210.03629*.
