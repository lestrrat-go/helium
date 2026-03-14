<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:my="http://my.net/" exclude-result-prefixes="my xs">

  <!-- PURPOSE initial template available in several modes of which one is the initial mode -->
  
  <xsl:variable name="temp"><vimble/></xsl:variable>

   <xsl:template match="/" mode="a b c">
    <out>
      <xsl:apply-templates select="$temp/*"/>
    </out>
  </xsl:template>
  
  <xsl:template match="*">
    <xsl:copy/>
  </xsl:template>



</xsl:stylesheet>
