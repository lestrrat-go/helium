<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/various"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test schema-element($name) as the first step in path expression. -->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/various"
      schema-location="variousTypesNodeTest.xsd"/>

   <t:template match="/my:userNode">
      <out>
         <E1>
            <t:value-of select="schema-element(my:complexMixedUserElem)/@age"/>
         </E1>
         <E2>
            <t:value-of select="schema-element(my:complexMixedUserElem)/company/name"/>
         </E2>
         <E3>
            <t:value-of select="schema-element(my:complexMixedUserElem)/company/name/name()"/>
         </E3>
      </out>
   </t:template>
</t:transform>
