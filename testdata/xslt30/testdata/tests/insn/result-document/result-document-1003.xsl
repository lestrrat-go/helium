<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test for generating two result trees with same URI where the first one is created 
				implicitly and the second with xsl:result-document without @href. -->

   <t:output method="xml"/>

   <t:template match="/">
      <out>First doc in implict xsl:result-document</out>
      <t:result-document>
         <out>Second doc in explicit xsl:result-document</out>
      </t:result-document>
   </t:template>
</t:transform>
