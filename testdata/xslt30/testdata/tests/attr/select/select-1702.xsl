<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:apply-templates with variable reference and predicate -->

   <t:variable name="var"> 
		    <t:copy-of select="((/doc/element())[1]/descendant::*,(/doc/element())[2]/descendant::*)[count(./preceding-sibling::*) eq 0][./descendant::element()]"/>
	  </t:variable>

   <t:template match="doc">
		    <out>
         <t:apply-templates select="$var[position() eq last()]"/>
      </out>
	  </t:template>

   <t:template match="@*">
		    <t:text>attr found: </t:text>
      <t:value-of select="name(.)"/>
      <t:text> | </t:text>
	  </t:template>

   <t:template match="*">
		    <t:text>elem found: </t:text>
      <t:value-of select="name(.)"/>
      <t:text> | </t:text>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
