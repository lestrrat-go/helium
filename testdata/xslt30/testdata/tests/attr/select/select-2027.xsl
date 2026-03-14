<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" exclude-result-prefixes="xs">
   <!-- Purpose: Test of select in xsl:for-each with sequence of node and atomic values and predicate -->

   <xsl:variable name="var" select="(xs:boolean('true'), 2.343, xs:gYearMonth('2005-10'), 3e5)"/>

   <xsl:template match="doc">
      <out>
         <t1>
            <xsl:for-each select="(sort(../child::element()/attribute()), (xs:boolean('true'), 2.343, xs:gYearMonth('2005-10'), 3e5) )">
               <xsl:value-of select="."/>
               <xsl:text>|</xsl:text>
            </xsl:for-each>
         </t1>
         <t2>
            <xsl:for-each select="(sort(../child::element()/attribute()), $var )[. castable as xs:integer] ">
               <xsl:value-of select="."/>
               <xsl:text>|</xsl:text>
            </xsl:for-each>
         </t2>
      </out>
   </xsl:template>

   <xsl:template match="text()"/>
</xsl:transform>
