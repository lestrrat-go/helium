<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" version="2.0" exclude-result-prefixes="xs">
   <!-- Purpose: Test of select in xsl:apply-templates with forExpr -->

   <xslt:variable name="var">
      <num1/>
   </xslt:variable>

   <xslt:template match="doc">
      <out>
         <t1>
            <xslt:apply-templates
               select="(for $rep in ./* return $rep/descendant-or-self::*)[$var] "/>
         </t1>
         <t2>
            <xslt:apply-templates
               select="(for $rep in ./* return $rep/descendant-or-self::*)[child::element()] "/>
         </t2>
      </out>
   </xslt:template>

   <xslt:template match="@*">
      <xslt:value-of select="name(.)"/>
      <xslt:text> | </xslt:text>
   </xslt:template>

   <xslt:template match="*">
      <xslt:value-of select="name(.)"/>
      <xslt:text> | </xslt:text>
   </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>
