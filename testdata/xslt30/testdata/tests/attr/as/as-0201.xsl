<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test with tunnel xsl:with-param and @as=empty-sequence(). 
  				The values for @select are () and empty sequence obtained at run time. -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:template match="/doc">
      <out>
         <xslt:apply-templates select="item1">
            <xslt:with-param name="par1" select="()" tunnel="yes" as="empty-sequence()"/>
            <xslt:with-param name="par2" select="item3" tunnel="yes" as="empty-sequence()"/>
         </xslt:apply-templates>
      </out>
   </xslt:template>

   <xslt:template match="item1" as="empty-sequence()">
      <xslt:apply-templates select="item2"/>
   </xslt:template>

   <xslt:template match="item2" as="empty-sequence()">
      <xslt:param name="par1" tunnel="yes"/>
      <xslt:param name="par2" tunnel="yes"/>
      <xslt:sequence select="$par1"/>
      <xslt:sequence select="$par2"/>
   </xslt:template>
</xslt:transform>
