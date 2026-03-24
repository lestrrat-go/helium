<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test of xsl:template which contains an attribute node from 
  				input file or xsl:attribute and has @as="attribute(QName, xs:untypedAtomic)". -->

   <xslt:template match="/">
      <out>
         <e1>
            <xslt:apply-templates select="doc/item/@attrib"/>
         </e1>
         <e2>
            <xslt:call-template name="a1"/>
         </e2>
      </out>
   </xslt:template>

   <xslt:template match="@attrib" as="attribute(attrib, xs:untypedAtomic)">
      <xslt:copy-of select="."/>
   </xslt:template>

   <xslt:template name="a1" as="attribute(e1, xs:untypedAtomic)">
      <xslt:attribute name="e1">attribute</xslt:attribute>
   </xslt:template>
</xslt:transform>
