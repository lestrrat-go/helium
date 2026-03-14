<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/various"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             input-type-annotations="strip"
             version="2.0">
<!-- Purpose: Show that stylesheet attribute @input-type-annotation="strip" makes the schema-element test fail. -->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/various"
                    schema-location="variousTypesNodeTest.xsd"/>

   <t:template match="/my:userNode">
	     <out>
         
         <E1>
            <t:value-of select="schema-element(my:simpleBuiltin)"/>
         </E1>
         
         <E2>
            <t:value-of select="schema-element(my:simpleUserElem)"/>
         </E2>
         
         <E3>
            <t:value-of select="schema-element(my:simpleUserList)"/>
         </E3>
         
         <E4>
            <t:value-of select="schema-element(my:simpleUserUnion)"/>
         </E4>
         
         <E5>
            <t:copy-of select="schema-element(my:complexUserElem)"/>
         </E5>
         
         <E6>
            <t:copy-of select="schema-element(my:complexMixedUserElem)"/>
         </E6>
         
         <E7>
            <t:value-of select="schema-element(my:complexSimpleContentElem)"/>
         </E7>
         
         <E8>
            <t:value-of select="schema-element(my:onlyAttributes)"/>
         </E8>
         
      </out>
   </t:template>
</t:transform>
