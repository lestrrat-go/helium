<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:apply-templates with element() and predicate index -->

   <t:template match="doc">
		    <out>
         <t:apply-templates select="(.//element())[position() ne last()][position() gt 1]"/>
      </out>
	  </t:template>

   <t:template match="*">
		    <t:value-of select="name(.)"/>
      <t:text>|</t:text>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
