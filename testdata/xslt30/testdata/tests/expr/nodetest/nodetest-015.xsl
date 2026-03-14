<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/various"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test schema-element($name) in the middle of a path expression. -->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/various"
      schema-location="variousTypesNodeTest.xsd"/>

   <t:template match="/">
      <out>
         <E1>
            <t:value-of select="my:userNode/schema-element(my:complexMixedUserElem)/@age"/>
         </E1>
         <E2>
            <t:value-of select="//my:userNode/schema-element(my:complexMixedUserElem)/company/name"/>
         </E2>
         <E3>
            <t:value-of select="my:userNode/schema-element(my:complexMixedUserElem)/company/name/name()"/>
         </E3>
      </out>
   </t:template>
</t:transform>
