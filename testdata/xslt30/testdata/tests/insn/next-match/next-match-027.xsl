<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
<!-- Purpose: Simple test of next-match where the match pattern is a NameTest -->

   <xsl:template name="xsl:initial-template">
      <out>
         <xsl:apply-templates select="1 to 5"/>
      </out>
   </xsl:template>
   
   <xsl:template match=".[. ge 5]" priority="5">E<xsl:next-match/></xsl:template>
   <xsl:template match=".[. ge 4]" priority="4">D<xsl:next-match/></xsl:template>
   <xsl:template match=".[. ge 3]" priority="3">C<xsl:next-match/></xsl:template>
   <xsl:template match=".[. ge 2]" priority="2">B<xsl:next-match/></xsl:template>
   <xsl:template match=".[. ge 1]" priority="1">A<xsl:next-match/></xsl:template>
   
</xsl:transform>
