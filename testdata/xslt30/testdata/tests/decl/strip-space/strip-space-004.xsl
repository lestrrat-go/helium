<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test xsl:strip-space with @elements="*:NCName". 
				Template rules have  match="*", match="*:NCName" and match="QName". -->

   <t:preserve-space elements="*"/>

   <t:strip-space elements="*:elem"/>

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

   <t:template match="*:elem">
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
