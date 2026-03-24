<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/various"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test schema-attribute($name) with a typed attribute created inside a variable.-->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/various"
                    schema-location="variousTypesNodeTest.xsd"/>
   
   <t:variable name="var" as="schema-element(my:userNode)">
      <my:userNode t:validation="strict">
         <t:attribute name="my:specialPart">123-AB</t:attribute>
         <my:complexSimpleContentElem my:count="ten" >
            10
         </my:complexSimpleContentElem>
      </my:userNode>
   </t:variable>

   <t:template match="/">
	     <out>
         <t:apply-templates select="$var"/>
      </out>
   </t:template>

   <t:template match="*">
	     <E1 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="schema-attribute(my:specialPart)"/>
      </E1>
   </t:template>
</t:transform>
