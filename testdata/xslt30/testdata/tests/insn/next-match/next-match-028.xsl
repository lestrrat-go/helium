<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
<!-- Purpose: tail call on next-match within a named template (Saxon bug 2818) -->
   
   <xsl:variable name="alpha">alpha</xsl:variable>
   <xsl:variable name="beta">beta</xsl:variable>
   <xsl:variable name="gamma">gamma</xsl:variable>
   <xsl:variable name="delta">delta</xsl:variable>
   
   <xsl:variable name="doc">
      <a>
         <b>2</b>
         <c>3</c>
         <d>4</d>
      </a>
   </xsl:variable>

   <xsl:template name="xsl:initial-template">
      <one>
         <xsl:apply-templates select="$doc//b"/>
      </one>
   </xsl:template>
   
   <xsl:template match="b">
      <two><xsl:call-template name="N"/></two>
   </xsl:template>
   
   <xsl:template name="N">
      <three/>
      <xsl:next-match/>
   </xsl:template>
   
   <xsl:template match="*">
      <xsl:copy-of select="$alpha, $beta, $gamma, $delta"/>
   </xsl:template>
   
   
</xsl:transform>
