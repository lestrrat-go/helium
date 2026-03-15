<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:apply-templates with predicate -->

   <t:template match="document-node()">
		    <out>
         <t:apply-templates select="self::node()/doc[position() eq last()]"/>
      </out>
	  </t:template>

   <t:template match="doc">
		    <t:value-of select="name(.)"/>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
