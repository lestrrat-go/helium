<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:test="http://www.test.org/Transform"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0"
             exclude-result-prefixes="test">
<!-- Purpose: Test of select in xsl:apply-templates with *:NCName -->

   <t:template match="doc">
		    <out>
         <t:apply-templates select="./*:num1"/>
      </out>
	  </t:template>

   <t:template match="*">
		    <t:value-of select="name(.)"/>
      <t:text>|</t:text>
		    <t:apply-templates select=".//test:*"/>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
