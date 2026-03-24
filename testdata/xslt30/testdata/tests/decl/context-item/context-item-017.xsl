<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <!-- Test xsl:context-item with next-match and use="absent" (dynamic error).-->
  
    <xsl:template match="doc" name="unused">
      <xsl:context-item use="absent"/>
      <xsl:copy-of select="."/>
    </xsl:template>
    
    <xsl:template match="doc" priority="12">
      <in><xsl:next-match/></in>
    </xsl:template>
    
    <xsl:template match="/">
      <out>
        <xsl:apply-templates select="doc"/> 
      </out>
    </xsl:template>  
          
  </xsl:stylesheet>