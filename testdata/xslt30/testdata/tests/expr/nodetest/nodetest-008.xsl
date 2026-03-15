<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/various"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test with schema-element($name) where $name is a top level element declaration. The type of the candidate 
  				node and the type in the element declaration in the schema match. Types tested are:
				-simple built-in, simple derived, list derived, union, complex with element only content,
				complex with mixed content, complex with simple content, complex with empty content -->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/various"
                    schema-location="variousTypesNodeTest.xsd"/>

   <t:template match="/my:userNode">
	     <out>
         <t:text>
</t:text>
         <E1>
            <t:value-of select="schema-element(my:simpleBuiltin)"/>
         </E1>
         <t:text>
</t:text>
         <E2>
            <t:value-of select="schema-element(my:simpleUserElem)"/>
         </E2>
         <t:text>
</t:text>
         <E3>
            <t:value-of select="schema-element(my:simpleUserList)"/>
         </E3>
         <t:text>
</t:text>
         <E4>
            <t:value-of select="schema-element(my:simpleUserUnion)"/>
         </E4>
         <t:text>
</t:text>
         <E5>
            <t:value-of select="schema-element(my:complexUserElem)/*[1]"/>
         </E5>
         <t:text>
</t:text>
         <E6>
            <t:value-of select="schema-element(my:complexMixedUserElem)/*[1]"/>
         </E6>
         <t:text>
</t:text>
         <E7>
            <t:value-of select="schema-element(my:complexSimpleContentElem)"/>
         </E7>
         <t:text>
</t:text>
         <E8>
            <t:value-of select="schema-element(my:onlyAttributes)"/>
         </E8>
         <t:text>
</t:text>
      </out>
   </t:template>
</t:transform>
