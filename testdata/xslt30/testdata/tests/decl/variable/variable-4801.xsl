<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
   
   <?spec xslt#variables?>
   <!-- Purpose: Tests xsl:variable with select comparing untyped id() -->
   
   <xsl:output method="xml" encoding="UTF-8" indent="no"/>
   
   <xsl:template match="/">
      <out>
         <xsl:apply-templates/>
      </out>
   </xsl:template>
   
   <xsl:template match="doc[@id='id0']">
      <xsl:variable name="p1" select="'id2'"/>
      <xsl:variable name="p2"
         select="'id4','id5','id8','id10','id11','id13','id14','id17'" />
      <xsl:variable name="p3" select="'id1','id9','id16','id18'"/>
      <xsl:variable name="p4" select=".//*[not(@id = ($p1, $p2, $p3))]/@id"/>
      <xsl:for-each select="$p4">
         <e>
            <xsl:value-of select="."/>
         </e>
      </xsl:for-each>
   </xsl:template>
   
</xsl:stylesheet>