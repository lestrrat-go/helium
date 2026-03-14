<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <!-- Test xsl:context-item with legal whitespace -->
  
    <xsl:template name="t" xml:space="preserve">
      <xsl:context-item use=" required " as=" xs:integer "/>
      <xsl:param name="p" select="3"/>
      <xsl:sequence select=". + $p"/>
    </xsl:template>
    
    <xsl:template name="main">
      <out>
        <xsl:for-each select="1 to 5">
          <xsl:call-template name="t"/>
        </xsl:for-each>  
      </out>
    </xsl:template>  
          
  </xsl:stylesheet>