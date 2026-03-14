<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
   <!-- Purpose: Test case to create a new element using xsl:element and use inherit-namespace="0". -->

   <t:template match="doc">
      <out>
         <t:element name="Outer" namespace="http://www.test.com">
            <t:element name="{data/e2}" inherit-namespaces="0">
               <t:sequence select="//e2"/>
            </t:element>
         </t:element>
      </out>
   </t:template>
</t:transform>
