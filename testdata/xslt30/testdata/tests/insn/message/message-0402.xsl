<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests using xsl:message which contains characters not in the standard ASCII range. -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="doc">
	     <out>
         <t:message terminate="no" select="'äb Äb ç € æ ð Ë Ñ ß Þ Ø Û'"/>
      </out>
	  </t:template>
</t:transform>
