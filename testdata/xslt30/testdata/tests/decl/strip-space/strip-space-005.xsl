<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://www.test1.example.org" xmlns:t="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="foo" version="2.0">
   <!-- Purpose: Test xsl:preserve-space with @elements="NCName:*". 
  				Template rules have  match="*" , match="NCName:* and match="QName". -->

   <t:strip-space elements="*"/>

   <t:preserve-space elements="foo:*"/>

   <t:template match="/doc">
      <out>
         <t:apply-templates/>
      </out>
   </t:template>

   <t:template match="elem">
      <a>
         <t:value-of select="."/>
      </a>
   </t:template>

   <t:template match="foo:*">
      <b>
         <t:value-of select="."/>
      </b>
   </t:template>

   <t:template match="*">
      <c>
         <t:value-of select="."/>
      </c>
   </t:template>
</t:transform>
