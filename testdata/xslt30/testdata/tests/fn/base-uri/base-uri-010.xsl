<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xml:base="http://www.baseuri.exmpl/tests/"
             version="2.0">
<!-- Purpose: Testcase with fn:static-base-uri to return the base-uri property which has been overriden several times by descendant elements.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/" xml:base="http://www.one.org/tests/">
      <out xml:base="http://www.two.org/tests/">
         <t:value-of xml:base="http://www.three.org/tests/" select="static-base-uri()"/>
      </out>
   </t:template>
</t:transform>
